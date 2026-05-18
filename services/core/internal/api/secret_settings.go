package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	secretSettingOwner        = "api.settings"
	secretSettingKeyReadLimit = 4096
)

func loadSecretSetting(ctx context.Context, db *sql.DB, dataDir, key, def string) string {
	plain, ok := loadSecretSettingVault(ctx, db, dataDir, key)
	if ok {
		return plain
	}
	return loadSetting(db, key, def)
}

func upsertSecretSetting(ctx context.Context, db *sql.DB, dataDir, key, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return deleteSecretSetting(ctx, db, key)
	}
	ciphertext, err := encryptSecretSetting(dataDir, value)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO secrets_vault(name, ciphertext, created_at, owner) VALUES(?,?,?,?) ON CONFLICT(name) DO UPDATE SET ciphertext=excluded.ciphertext, owner=excluded.owner`, secretSettingVaultName(key), ciphertext, time.Now().UnixMilli(), secretSettingOwner); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}

func deleteSecretSetting(ctx context.Context, db *sql.DB, key string) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM secrets_vault WHERE name = ?`, secretSettingVaultName(key)); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}

func loadSecretSettingVault(ctx context.Context, db *sql.DB, dataDir, key string) (string, bool) {
	if db == nil {
		return "", false
	}
	var ciphertext string
	err := db.QueryRowContext(ctx, `SELECT ciphertext FROM secrets_vault WHERE name = ?`, secretSettingVaultName(key)).Scan(&ciphertext)
	if err != nil {
		return "", false
	}
	plain, err := decryptSecretSetting(dataDir, ciphertext)
	if err != nil {
		return "", false
	}
	return plain, true
}

func secretSettingVaultName(key string) string {
	return "settings." + strings.TrimSpace(key)
}

func encryptSecretSetting(dataDir, value string) (string, error) {
	key, err := secretSettingEncryptionKey(dataDir)
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

func decryptSecretSetting(dataDir, ciphertext string) (string, error) {
	key, err := secretSettingEncryptionKey(dataDir)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertext))
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

func secretSettingEncryptionKey(dataDir string) ([]byte, error) {
	keyPath := strings.TrimSpace(os.Getenv("FORGE_ENCRYPTION_KEY_PATH"))
	if keyPath == "" {
		keyPath = filepath.Join(nonEmptySecretDataDir(dataDir), "secrets", "master.key")
	}
	if b, err := readSecretSettingFileBounded(keyPath, secretSettingKeyReadLimit); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
		return nil, fmt.Errorf("invalid encryption key at %s", keyPath)
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

func readSecretSettingFileBounded(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("secret setting key file exceeds %d bytes", maxBytes)
	}
	return b, nil
}

func nonEmptySecretDataDir(dataDir string) string {
	if strings.TrimSpace(dataDir) != "" {
		return dataDir
	}
	return os.TempDir()
}
