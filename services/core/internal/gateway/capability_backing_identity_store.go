package gateway

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (t *capabilityBackingTool) encryptionKey() ([]byte, error) {
	keyPath := strings.TrimSpace(os.Getenv("FORGE_ENCRYPTION_KEY_PATH"))
	if keyPath == "" {
		keyPath = filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "secrets", "master.key")
	}
	if b, err := readCapabilityFileBounded(keyPath, "encryption key file", gatewayEncryptionKeyReadLimit); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func (t *capabilityBackingTool) encryptSecret(value string) (string, error) {
	value, err := normalizeSecretPlaintext(value)
	if err != nil {
		return "", err
	}
	key, err := t.encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

func (t *capabilityBackingTool) decryptSecret(ciphertext string) (string, error) {
	ciphertext, err := normalizeSecretCiphertext(ciphertext)
	if err != nil {
		return "", err
	}
	key, err := t.encryptionKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (t *capabilityBackingTool) storeSecret(ctx context.Context, name, ciphertext string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("secret name required")
	}
	return t.execDB(ctx, `INSERT INTO secrets_vault(name, ciphertext, created_at, owner) VALUES(?,?,?,?) ON CONFLICT(name) DO UPDATE SET ciphertext=excluded.ciphertext, owner=excluded.owner`, name, ciphertext, time.Now().UnixMilli(), "gateway")
}

func (t *capabilityBackingTool) loadSecret(ctx context.Context, name string) (string, error) {
	return t.scalarSettingOrTable(ctx, `SELECT ciphertext FROM secrets_vault WHERE name = ?`, name)
}

func (t *capabilityBackingTool) signOrVerify(req Request) (Result, error) {
	seed := sha256.Sum256([]byte(nonEmpty(os.Getenv("FORGE_IDENTITY_SIGNING_SEED"), "forge-local-signing-key")))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	message, err := normalizeIdentityMessage(inputString(req.Input, "message"))
	if err != nil {
		return Result{}, err
	}
	if t.capability.Name == "sign" {
		sig := ed25519.Sign(priv, message)
		return capabilityOK("message signed", map[string]any{"publicKey": base64.StdEncoding.EncodeToString(pub), "signature": base64.StdEncoding.EncodeToString(sig)}), nil
	}
	signature, err := normalizeIdentitySignature(inputString(req.Input, "signature"))
	if err != nil {
		return Result{}, err
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return Result{}, err
	}
	return capabilityOK("signature verified", map[string]any{"valid": ed25519.Verify(pub, message, sig)}), nil
}

func (t *capabilityBackingTool) settingGet(ctx context.Context, key string) (string, error) {
	if t.db == nil {
		return "", errors.New("db unavailable")
	}
	var value string
	err := t.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

func (t *capabilityBackingTool) settingSet(ctx context.Context, key, value string) error {
	if t.db == nil {
		return errors.New("db unavailable")
	}
	_, err := t.db.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (t *capabilityBackingTool) execDB(ctx context.Context, query string, args ...any) error {
	if t.db == nil {
		return errors.New("db unavailable")
	}
	_, err := t.db.ExecContext(ctx, query, args...)
	return err
}

func (t *capabilityBackingTool) scalarSettingOrTable(ctx context.Context, query string, args ...any) (string, error) {
	if t.db == nil {
		return "", errors.New("db unavailable")
	}
	var value string
	err := t.db.QueryRowContext(ctx, query, args...).Scan(&value)
	return value, err
}

func (t *capabilityBackingTool) queryRows(ctx context.Context, query string, args ...any) (Result, error) {
	if t.db == nil {
		return Result{}, errors.New("db unavailable")
	}
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}
	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return Result{}, err
		}
		row := map[string]any{}
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			default:
				row[col] = v
			}
		}
		out = append(out, row)
	}
	return capabilityOK("query completed", map[string]any{"rows": out, "count": len(out)}), rows.Err()
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
