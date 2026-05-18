package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (t *capabilityBackingTool) runConfiguredCommand(ctx context.Context, req Request, envName, fallback string) (Result, error) {
	bin := nonEmpty(os.Getenv(envName), fallback)
	args, err := normalizeConfiguredCommandArgs(inputString(req.Input, "args"))
	if err != nil {
		return Result{}, err
	}
	return runCommand(ctx, t.workspace, bin, args...)
}

func runCommand(ctx context.Context, dir, bin string, args ...string) (Result, error) {
	if _, err := exec.LookPath(bin); err != nil {
		return Result{}, fmt.Errorf("%s not found: %w", bin, err)
	}
	cmd := newGatewayCommand(ctx, dir, bin, args...)
	out, err := boundedCombinedOutput(cmd)
	return capabilityOK("command completed", map[string]any{"command": append([]string{bin}, args...), "output": out}), err
}

func fetchURL(ctx context.Context, rawURL, method string) (Result, error) {
	if rawURL == "" {
		return Result{}, errors.New("url is required")
	}
	if method == "" {
		method = http.MethodGet
	}
	parsed, err := validateOutboundHTTPURL(ctx, rawURL, nil)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, method, parsed.String(), nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := newGuardedOutboundHTTPClient(ctx, nil).Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := readGatewayHTTPResponseBody(resp.Body, "capability http request", gatewayCapabilityHTTPResponseBodyLimit)
	if err != nil {
		return Result{}, err
	}
	return capabilityOK("http request completed", map[string]any{"status": resp.Status, "statusCode": resp.StatusCode, "body": string(body)}), nil
}

func defaultTTSCommand() string {
	if runtime.GOOS == "darwin" {
		return "say"
	}
	return "espeak-ng"
}

func (t *capabilityBackingTool) callDesktopBridge(ctx context.Context, req Request) (Result, error) {
	port := strings.TrimSpace(os.Getenv("FORGE_DESKTOP_BRIDGE_PORT"))
	token := strings.TrimSpace(os.Getenv("FORGE_DESKTOP_BRIDGE_TOKEN"))
	if port == "" || token == "" {
		return Result{}, errors.New("desktop bridge requires FORGE_DESKTOP_BRIDGE_PORT and FORGE_DESKTOP_BRIDGE_TOKEN")
	}
	body, err := t.marshalDesktopBridgePayload(req)
	if err != nil {
		return Result{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:"+port+"/forge/desktop-bridge/tool", strings.NewReader(string(body)))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, err := readGatewayHTTPResponseBody(resp.Body, "desktop bridge", gatewayDesktopBridgeResponseBodyLimit)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("desktop bridge returned %s: %s", resp.Status, string(respBody))
	}
	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		payload = map[string]any{"body": string(respBody)}
	}
	return capabilityOK("desktop bridge completed", payload), nil
}

func (t *capabilityBackingTool) marshalDesktopBridgePayload(req Request) (string, error) {
	body, err := json.Marshal(map[string]any{"capability": t.capability.ID, "input": nonNilMap(req.Input), "paths": req.Paths})
	if err != nil {
		return "", err
	}
	if len(body) > maxDesktopBridgeRequestBodyBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errDesktopBridgeRequestBodyTooLarge, len(body), maxDesktopBridgeRequestBodyBytes)
	}
	return string(body), nil
}
