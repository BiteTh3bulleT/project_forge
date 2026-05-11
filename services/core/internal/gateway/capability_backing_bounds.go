package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	gatewayProcStatusReadLimit             = 64 << 10
	gatewayWorkspaceSearchReadLimit        = 256 << 10
	gatewayEncryptionKeyReadLimit          = 4 << 10
	maxCodeEvalInputBytes                  = 128 << 10
	maxTestOutputInputBytes                = 512 << 10
	maxCapabilityMemoryTextBytes           = 256 << 10
	maxCapabilityMemoryIDBytes             = 512
	maxCapabilityMemoryPayloadBytes        = 768 << 10
	maxConfiguredCommandArgsBytes          = 64 << 10
	maxIdentityMessageBytes                = 256 << 10
	maxIdentitySignatureBytes              = 4 << 10
	maxCapabilitySearchQueryBytes          = 8 << 10
	maxSecretPlaintextBytes                = 64 << 10
	maxSecretCiphertextBytes               = 128 << 10
	maxCapabilityConfigKeyBytes            = 512
	maxCapabilityConfigValueBytes          = 64 << 10
	maxIdentityTokenInputBytes             = 8 << 10
	maxIdentityTokenIDBytes                = 256
	maxAlertRuleNameBytes                  = 512
	maxAlertRuleExpressionBytes            = 8 << 10
	maxScheduleIDBytes                     = 512
	maxSchedulePayloadBytes                = 64 << 10
	maxDesktopBridgeRequestBodyBytes       = 64 << 10
	maxCapabilityResultInputFields         = 32
	maxCapabilityResultInputFieldNameBytes = 128
	maxCapabilityResultInputSummaryBytes   = 8 << 10
)

var errCodeEvalTooLarge = errors.New("code.eval_code input too large")

var errTestOutputTooLarge = errors.New("code.parse_test_results output too large")

var errCapabilityMemoryTextTooLarge = errors.New("capability memory text too large")

var errCapabilityMemoryIDTooLarge = errors.New("capability memory id too large")

var errCapabilityMemoryPayloadTooLarge = errors.New("capability memory payload too large")

var errConfiguredCommandArgsTooLarge = errors.New("configured command args too large")

var errIdentityMessageTooLarge = errors.New("identity message too large")

var errIdentitySignatureTooLarge = errors.New("identity signature too large")

var errCapabilitySearchQueryTooLarge = errors.New("capability search query too large")

var errSecretPlaintextTooLarge = errors.New("secret plaintext too large")

var errSecretCiphertextTooLarge = errors.New("secret ciphertext too large")

var errCapabilityConfigKeyTooLarge = errors.New("capability config key too large")

var errCapabilityConfigValueTooLarge = errors.New("capability config value too large")

var errIdentityTokenInputTooLarge = errors.New("identity token input too large")

var errIdentityTokenIDTooLarge = errors.New("identity token id too large")

var errAlertRuleNameTooLarge = errors.New("alert rule name too large")

var errAlertRuleExpressionTooLarge = errors.New("alert rule expression too large")

var errScheduleIDTooLarge = errors.New("schedule id too large")

var errSchedulePayloadTooLarge = errors.New("schedule payload too large")

var errDesktopBridgeRequestBodyTooLarge = errors.New("desktop bridge request body too large")

func readCapabilityFileBounded(path, label string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s read limit must be positive", label)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Mode().IsRegular() && info.Size() > limit {
		return nil, fmt.Errorf("%s too large: %d bytes exceeds %d byte limit", label, info.Size(), limit)
	}
	body, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s too large: exceeds %d byte limit", label, limit)
	}
	return body, nil
}

func normalizeCodeEvalInput(raw string) (string, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		return "", errors.New("code.eval_code requires input.code")
	}
	if len(code) > maxCodeEvalInputBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCodeEvalTooLarge, len(code), maxCodeEvalInputBytes)
	}
	return code, nil
}

func normalizeTestOutputInput(raw string) (string, error) {
	output := strings.TrimSpace(raw)
	if len(output) > maxTestOutputInputBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errTestOutputTooLarge, len(output), maxTestOutputInputBytes)
	}
	return output, nil
}

func normalizeCapabilityMemoryText(raw, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if len(value) > maxCapabilityMemoryTextBytes {
		return "", fmt.Errorf("%w: %s %d > %d bytes", errCapabilityMemoryTextTooLarge, field, len(value), maxCapabilityMemoryTextBytes)
	}
	return value, nil
}

func normalizeCapabilityMemoryID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if len(id) > maxCapabilityMemoryIDBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCapabilityMemoryIDTooLarge, len(id), maxCapabilityMemoryIDBytes)
	}
	return id, nil
}

func marshalCapabilityMemoryPayload(input map[string]any) (string, error) {
	payload, err := json.Marshal(nonNilMap(input))
	if err != nil {
		return "", err
	}
	if len(payload) > maxCapabilityMemoryPayloadBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCapabilityMemoryPayloadTooLarge, len(payload), maxCapabilityMemoryPayloadBytes)
	}
	return string(payload), nil
}

func normalizeConfiguredCommandArgs(raw string) ([]string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return nil, nil
	}
	if len(normalized) > maxConfiguredCommandArgsBytes {
		return nil, fmt.Errorf("%w: %d > %d bytes", errConfiguredCommandArgsTooLarge, len(normalized), maxConfiguredCommandArgsBytes)
	}
	return strings.Fields(normalized), nil
}

func normalizeIdentityMessage(raw string) ([]byte, error) {
	message := []byte(strings.TrimSpace(raw))
	if len(message) > maxIdentityMessageBytes {
		return nil, fmt.Errorf("%w: %d > %d bytes", errIdentityMessageTooLarge, len(message), maxIdentityMessageBytes)
	}
	return message, nil
}

func normalizeIdentitySignature(raw string) (string, error) {
	signature := strings.TrimSpace(raw)
	if len(signature) > maxIdentitySignatureBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errIdentitySignatureTooLarge, len(signature), maxIdentitySignatureBytes)
	}
	return signature, nil
}

func normalizeCapabilitySearchQuery(raw, label string) (string, error) {
	query := strings.TrimSpace(raw)
	if len(query) > maxCapabilitySearchQueryBytes {
		return "", fmt.Errorf("%w: %s %d > %d bytes", errCapabilitySearchQueryTooLarge, label, len(query), maxCapabilitySearchQueryBytes)
	}
	return query, nil
}

func normalizeSecretPlaintext(raw string) (string, error) {
	if len(raw) > maxSecretPlaintextBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errSecretPlaintextTooLarge, len(raw), maxSecretPlaintextBytes)
	}
	return raw, nil
}

func normalizeSecretCiphertext(raw string) (string, error) {
	ciphertext := strings.TrimSpace(raw)
	if len(ciphertext) > maxSecretCiphertextBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errSecretCiphertextTooLarge, len(ciphertext), maxSecretCiphertextBytes)
	}
	return ciphertext, nil
}

func normalizeCapabilityConfigKey(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if len(key) > maxCapabilityConfigKeyBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCapabilityConfigKeyTooLarge, len(key), maxCapabilityConfigKeyBytes)
	}
	return key, nil
}

func normalizeCapabilityConfigValue(raw string) (string, error) {
	if len(raw) > maxCapabilityConfigValueBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCapabilityConfigValueTooLarge, len(raw), maxCapabilityConfigValueBytes)
	}
	return raw, nil
}

func normalizeIdentityTokenInput(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if len(token) > maxIdentityTokenInputBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errIdentityTokenInputTooLarge, len(token), maxIdentityTokenInputBytes)
	}
	return token, nil
}

func normalizeIdentityTokenID(raw string) (string, error) {
	tokenID := strings.TrimSpace(raw)
	if len(tokenID) > maxIdentityTokenIDBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errIdentityTokenIDTooLarge, len(tokenID), maxIdentityTokenIDBytes)
	}
	return tokenID, nil
}

func normalizeAlertRuleName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if len(name) > maxAlertRuleNameBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errAlertRuleNameTooLarge, len(name), maxAlertRuleNameBytes)
	}
	return name, nil
}

func normalizeAlertRuleExpression(raw string) (string, error) {
	expression := strings.TrimSpace(raw)
	if len(expression) > maxAlertRuleExpressionBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errAlertRuleExpressionTooLarge, len(expression), maxAlertRuleExpressionBytes)
	}
	return expression, nil
}

func normalizeScheduleID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if len(id) > maxScheduleIDBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errScheduleIDTooLarge, len(id), maxScheduleIDBytes)
	}
	return id, nil
}

func marshalSchedulePayload(input map[string]any) (string, error) {
	payload, err := json.Marshal(nonNilMap(input))
	if err != nil {
		return "", err
	}
	if len(payload) > maxSchedulePayloadBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errSchedulePayloadTooLarge, len(payload), maxSchedulePayloadBytes)
	}
	return string(payload), nil
}
