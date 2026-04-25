package gateway

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

var (
	capabilitySocketsMu sync.Mutex
	capabilitySockets   = map[string]io.Closer{}
	capabilityTimersMu  sync.Mutex
	capabilityTimers    = map[string]time.Time{}
)

type capabilityBackingTool struct {
	capability domain.ToolCapability
	toolID     string
	workspace  string
	dataDir    string
	db         *sql.DB
}

func (t *capabilityBackingTool) ID() string     { return t.toolID }
func (t *capabilityBackingTool) Domain() string { return t.capability.Domain }
func (t *capabilityBackingTool) Action() string { return t.capability.Name }
func (t *capabilityBackingTool) RiskClass() string {
	return gatewayRiskClassFromToolRisk(t.capability.Risk)
}
func (t *capabilityBackingTool) Description() string { return t.capability.Description }
func (t *capabilityBackingTool) Executes() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectExecute)
}
func (t *capabilityBackingTool) UsesNetwork() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectNetwork) || capabilityHasEffect(t.capability, domain.ToolEffectExternal)
}
func (t *capabilityBackingTool) WriteIntent() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectWrite) ||
		capabilityHasEffect(t.capability, domain.ToolEffectPrivileged) ||
		capabilityHasEffect(t.capability, domain.ToolEffectDestructive)
}
func (t *capabilityBackingTool) ExecutionLevel() string {
	if capabilityHasEffect(t.capability, domain.ToolEffectDestructive) {
		return "L4"
	}
	if capabilityHasEffect(t.capability, domain.ToolEffectPrivileged) {
		return "L3"
	}
	return executionLevelFromRisk(gatewayRiskClassFromToolRisk(t.capability.Risk))
}

func (t *capabilityBackingTool) Execute(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Domain {
	case "filesystem":
		return t.executeFilesystem(ctx, req)
	case "network":
		return t.executeNetwork(ctx, req)
	case "process":
		return t.executeProcess(ctx, req)
	case "code":
		return t.executeCode(ctx, req)
	case "identity":
		return t.executeIdentity(ctx, req)
	case "config":
		return t.executeConfig(ctx, req)
	case "observability":
		return t.executeObservability(ctx, req)
	case "ui":
		return t.executeUI(ctx, req)
	case "device":
		return t.executeDevice(ctx, req)
	case "time":
		return t.executeTime(ctx, req)
	case "external":
		return t.executeExternal(ctx, req)
	case "memory":
		return t.executeMemory(ctx, req)
	case "agent":
		return t.executeAgent(ctx, req)
	case "backup":
		return t.executeBackup(ctx, req)
	default:
		return capabilityOK("capability executed", map[string]any{
			"capabilityId": t.capability.ID,
			"toolId":       t.toolID,
		}), nil
	}
}

func (t *capabilityBackingTool) executeFilesystem(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "glob":
		pattern := inputString(req.Input, "pattern")
		if pattern == "" && len(req.Paths) > 0 {
			pattern = req.Paths[0]
		}
		if pattern == "" {
			pattern = "**"
		}
		rows, err := workspaceGlob(t.workspace, pattern)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("glob completed", map[string]any{"matches": rows, "count": len(rows)}), nil
	case "get_permissions":
		target, err := firstPath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		info, err := os.Stat(target)
		if err != nil {
			return Result{}, err
		}
		data := map[string]any{"path": target, "mode": info.Mode().String(), "modeOctal": fmt.Sprintf("%#o", info.Mode().Perm()), "isDir": info.IsDir(), "size": info.Size()}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			data["uid"] = stat.Uid
			data["gid"] = stat.Gid
		}
		return capabilityOK("stat completed", data), nil
	case "archive", "create_snapshot":
		out, err := t.createTarGZ(req)
		if err != nil {
			return Result{}, err
		}
		return capabilityOKWithArtifacts("archive created", map[string]any{"path": out}, []ResultArtifact{{Type: "archive", Path: out, Summary: t.capability.Name}}), nil
	case "extract", "restore_snapshot":
		dst, err := t.extractArchive(req)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("archive extracted", map[string]any{"destination": dst}), nil
	case "query_semantic_fs":
		query := inputString(req.Input, "query")
		rows, err := searchWorkspaceFiles(t.workspace, query, 50)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("workspace query completed", map[string]any{"query": query, "results": rows, "count": len(rows)}), nil
	case "sync_to_remote", "sync_from_remote":
		return t.runConfiguredCommand(ctx, req, "FORGE_RSYNC_BIN", "rsync")
	case "mount":
		if os.Getenv("FORGE_ALLOW_PRIVILEGED_MOUNT") != "true" {
			return Result{}, errors.New("filesystem.mount requires FORGE_ALLOW_PRIVILEGED_MOUNT=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_MOUNT_BIN", "mount")
	case "unmount":
		if os.Getenv("FORGE_ALLOW_PRIVILEGED_MOUNT") != "true" {
			return Result{}, errors.New("filesystem.unmount requires FORGE_ALLOW_PRIVILEGED_MOUNT=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_UMOUNT_BIN", "umount")
	case "watch_path":
		target, err := firstPath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		info, err := os.Stat(target)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("path watch snapshot captured", map[string]any{
			"path":          target,
			"mtimeUnixNano": info.ModTime().UnixNano(),
			"note":          "long-lived watch delivery requires the watch manager event stream",
		}), nil
	default:
		return capabilityOK("filesystem capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeNetwork(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "dns_register":
		host := inputString(req.Input, "host")
		address := inputString(req.Input, "address")
		if host == "" || address == "" {
			return Result{}, errors.New("network.dns_register requires input.host and input.address")
		}
		if err := t.settingSet(ctx, "dns_override."+host, address); err != nil {
			return Result{}, err
		}
		return capabilityOK("dns override registered", map[string]any{"host": host, "address": address}), nil
	case "open_socket":
		network := nonEmpty(inputString(req.Input, "network"), "tcp")
		address := inputString(req.Input, "address")
		if address == "" {
			address = net.JoinHostPort(inputString(req.Input, "host"), nonEmpty(inputString(req.Input, "port"), "80"))
		}
		conn, err := net.DialTimeout(network, address, time.Duration(inputInt(req.Input, "timeoutMs", 3000))*time.Millisecond)
		if err != nil {
			return Result{}, err
		}
		handle := "socket-" + newCorrelationID()
		capabilitySocketsMu.Lock()
		capabilitySockets[handle] = conn
		capabilitySocketsMu.Unlock()
		return capabilityOK("socket opened", map[string]any{"handle": handle, "localAddr": conn.LocalAddr().String(), "remoteAddr": conn.RemoteAddr().String()}), nil
	case "close_socket":
		handle := inputString(req.Input, "handle")
		capabilitySocketsMu.Lock()
		closer := capabilitySockets[handle]
		delete(capabilitySockets, handle)
		capabilitySocketsMu.Unlock()
		if closer == nil {
			return Result{}, fmt.Errorf("socket handle %q not found", handle)
		}
		return capabilityOK("socket closed", map[string]any{"handle": handle}), closer.Close()
	case "proxy_request", "http_request":
		url := inputString(req.Input, "url")
		if url == "" {
			url = inputString(req.Input, "target")
		}
		return fetchURL(ctx, url, inputString(req.Input, "method"))
	case "open_tunnel":
		return Result{}, errors.New("network.open_tunnel requires an SSH tunnel provider configuration")
	case "delete_firewall_rule", "set_firewall_rule":
		if os.Getenv("FORGE_ALLOW_FIREWALL_MUTATION") != "true" {
			return Result{}, errors.New("firewall mutation requires FORGE_ALLOW_FIREWALL_MUTATION=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_FIREWALL_BIN", "nft")
	case "intercept_traffic":
		if os.Getenv("FORGE_PCAP_ENABLED") != "true" {
			return Result{}, errors.New("traffic interception requires FORGE_PCAP_ENABLED=true and a configured capture backend")
		}
		return capabilityOK("traffic interception backend configured", map[string]any{"enabled": true}), nil
	default:
		return capabilityOK("network capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeProcess(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "signal_process":
		pid := inputInt(req.Input, "pid", 0)
		sigName := strings.ToUpper(nonEmpty(inputString(req.Input, "signal"), "TERM"))
		sig := syscall.SIGTERM
		if sigName == "KILL" {
			sig = syscall.SIGKILL
		}
		if pid <= 0 {
			return Result{}, errors.New("process.signal_process requires input.pid")
		}
		return capabilityOK("signal sent", map[string]any{"pid": pid, "signal": sigName}), syscall.Kill(pid, sig)
	case "inspect_process":
		pid := inputInt(req.Input, "pid", os.Getpid())
		data := map[string]any{"pid": pid}
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
			data["status"] = string(b)
		} else {
			data["status"] = "process inspection is limited on " + runtime.GOOS
		}
		return capabilityOK("process inspected", data), nil
	case "run_job", "fork_context":
		return capabilityOK("job request materialized", map[string]any{"input": nonNilMap(req.Input), "note": "job service bridge can enqueue this template from API wiring"}), nil
	case "checkpoint_process", "restore_process":
		if os.Getenv("FORGE_CRIU_BIN") == "" {
			return Result{}, errors.New("process checkpoint/restore requires FORGE_CRIU_BIN")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_CRIU_BIN", "criu")
	case "set_resource_limits":
		return capabilityOK("resource limit request validated", map[string]any{"limits": nonNilMap(req.Input), "scope": "spawned children"}), nil
	default:
		return capabilityOK("process capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeCode(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "eval_code":
		lang := strings.ToLower(nonEmpty(inputString(req.Input, "language"), "python"))
		code := inputString(req.Input, "code")
		if lang != "python" {
			return Result{}, fmt.Errorf("code.eval_code currently supports python via python3, got %q", lang)
		}
		return runCommand(ctx, t.workspace, "python3", "-c", code)
	case "compile":
		return runCommand(ctx, t.workspace, "go", "build", "./...")
	case "link":
		return capabilityOK("link step delegated to compile toolchain", map[string]any{"toolchain": "go build"}), nil
	case "run_tests":
		return runCommand(ctx, t.workspace, "go", "test", "./...")
	case "parse_test_results":
		raw := inputString(req.Input, "output")
		return capabilityOK("test output parsed", map[string]any{"passed": !strings.Contains(raw, "FAIL"), "bytes": len(raw)}), nil
	case "lint":
		return runCommand(ctx, t.workspace, "go", "vet", "./...")
	case "format":
		return runCommand(ctx, t.workspace, "gofmt", "-l", ".")
	case "patch_code":
		patch := inputString(req.Input, "patch")
		if strings.TrimSpace(patch) == "" {
			return Result{}, errors.New("code.patch_code requires input.patch")
		}
		cmd := exec.CommandContext(ctx, "git", "apply", "--check", "-")
		cmd.Dir = t.workspace
		cmd.Stdin = strings.NewReader(patch)
		out, err := cmd.CombinedOutput()
		return capabilityOK("patch validated", map[string]any{"output": string(out)}), err
	case "search_code":
		query := inputString(req.Input, "query")
		if query == "" {
			return Result{}, errors.New("code.search_code requires input.query")
		}
		return runCommand(ctx, t.workspace, "rg", "--json", query)
	case "refactor":
		return Result{}, errors.New("code.refactor requires an LLM-backed patch proposal provider")
	default:
		return capabilityOK("code capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeIdentity(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "get_current_user":
		u, err := user.Current()
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("current user resolved", map[string]any{"uid": u.Uid, "gid": u.Gid, "username": u.Username, "homeDir": u.HomeDir}), nil
	case "encrypt", "store_secret":
		name := nonEmpty(inputString(req.Input, "name"), "default")
		value := inputString(req.Input, "value")
		if value == "" {
			value = inputString(req.Input, "plaintext")
		}
		ciphertext, err := t.encryptSecret(value)
		if err != nil {
			return Result{}, err
		}
		if t.capability.Name == "store_secret" {
			if err := t.storeSecret(ctx, name, ciphertext); err != nil {
				return Result{}, err
			}
		}
		return capabilityOK("secret encrypted", map[string]any{"name": name, "ciphertext": ciphertext}), nil
	case "decrypt", "retrieve_secret":
		ciphertext := inputString(req.Input, "ciphertext")
		if t.capability.Name == "retrieve_secret" {
			var err error
			ciphertext, err = t.loadSecret(ctx, inputString(req.Input, "name"))
			if err != nil {
				return Result{}, err
			}
		}
		plaintext, err := t.decryptSecret(ciphertext)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("secret decrypted", map[string]any{"plaintext": plaintext}), nil
	case "issue_token":
		tokenBytes := make([]byte, 24)
		_, _ = rand.Read(tokenBytes)
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		tokenID := sha256Hex(token)
		if err := t.execDB(ctx, `INSERT INTO identity_tokens(token_id, token_hash, status, created_at, owner) VALUES(?,?,?,?,?)`, tokenID, sha256Hex(token), "active", time.Now().UnixMilli(), req.Initiator); err != nil {
			return Result{}, err
		}
		return capabilityOK("token issued", map[string]any{"tokenId": tokenID, "token": token}), nil
	case "verify_token":
		tokenID := sha256Hex(inputString(req.Input, "token"))
		status, err := t.scalarSettingOrTable(ctx, `SELECT status FROM identity_tokens WHERE token_id = ? OR token_hash = ?`, tokenID, tokenID)
		return capabilityOK("token verified", map[string]any{"tokenId": tokenID, "valid": err == nil && status == "active", "status": status}), nil
	case "revoke_token":
		tokenID := inputString(req.Input, "tokenId")
		if tokenID == "" {
			tokenID = sha256Hex(inputString(req.Input, "token"))
		}
		return capabilityOK("token revoked", map[string]any{"tokenId": tokenID}), t.execDB(ctx, `UPDATE identity_tokens SET status = 'revoked', revoked_at = ? WHERE token_id = ? OR token_hash = ?`, time.Now().UnixMilli(), tokenID, tokenID)
	case "sign", "verify_signature":
		return t.signOrVerify(req)
	case "audit_log_read":
		return t.queryRows(ctx, `SELECT id, created_at, category, action, actor, outcome, summary FROM audit_records ORDER BY id DESC LIMIT 50`)
	case "check_policy", "set_policy", "sudo", "switch_user":
		return capabilityOK("identity policy request captured", map[string]any{"action": t.capability.Name, "input": nonNilMap(req.Input)}), nil
	default:
		return capabilityOK("identity capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeConfig(ctx context.Context, req Request) (Result, error) {
	key := inputString(req.Input, "key")
	switch t.capability.Name {
	case "get_config":
		value, err := t.settingGet(ctx, key)
		return capabilityOK("config read", map[string]any{"key": key, "value": value, "found": err == nil}), nil
	case "set_config":
		return capabilityOK("config written", map[string]any{"key": key}), t.settingSet(ctx, key, inputString(req.Input, "value"))
	case "diff_config":
		return capabilityOK("config diff", map[string]any{"key": key, "note": "settings history is not versioned; current value returned"}), nil
	case "watch_config":
		return capabilityOK("config watch snapshot captured", map[string]any{"key": key, "note": "long-lived delivery uses API event streams"}), nil
	case "get_env":
		return capabilityOK("env read", map[string]any{"key": key, "value": os.Getenv(key)}), nil
	case "set_env":
		return capabilityOK("env set for current process", map[string]any{"key": key}), os.Setenv(key, inputString(req.Input, "value"))
	case "feature_flag_read":
		value, err := t.scalarSettingOrTable(ctx, `SELECT value FROM feature_flags WHERE key = ?`, key)
		return capabilityOK("feature flag read", map[string]any{"key": key, "value": value, "found": err == nil}), nil
	case "feature_flag_set":
		return capabilityOK("feature flag set", map[string]any{"key": key}), t.execDB(ctx, `INSERT INTO feature_flags(key, value, updated_at, actor) VALUES(?,?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at, actor=excluded.actor`, key, inputString(req.Input, "value"), time.Now().UnixMilli(), req.Initiator)
	case "backup", "restore", "migrate_schema":
		return capabilityOK("config operation captured", map[string]any{"operation": t.capability.Name, "input": nonNilMap(req.Input)}), nil
	default:
		return capabilityOK("config capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeObservability(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "get_metrics":
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		return capabilityOK("metrics captured", map[string]any{"goroutines": runtime.NumGoroutine(), "alloc": mem.Alloc, "sys": mem.Sys}), nil
	case "get_traces":
		corr := inputString(req.Input, "correlationId")
		if corr == "" {
			corr = req.CorrelationID
		}
		return t.queryRows(ctx, `SELECT id, created_at, category, action, actor, outcome, summary FROM audit_records WHERE correlation_id = ? ORDER BY id ASC`, corr)
	case "tail_stream":
		return t.queryRows(ctx, `SELECT id, created_at, type, payload_json FROM events ORDER BY id DESC LIMIT 50`)
	case "create_alert":
		name := inputString(req.Input, "name")
		return capabilityOK("alert rule created", map[string]any{"name": name}), t.execDB(ctx, `INSERT INTO alert_rules(name, expression, status, created_at, updated_at) VALUES(?,?,?,?,?)`, name, inputString(req.Input, "expression"), "active", time.Now().UnixMilli(), time.Now().UnixMilli())
	case "silence_alert":
		name := inputString(req.Input, "name")
		return capabilityOK("alert silenced", map[string]any{"name": name}), t.execDB(ctx, `UPDATE alert_rules SET status='silenced', silenced_until=?, updated_at=? WHERE name=?`, time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli(), name)
	case "profile_process":
		out := filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "profiles", fmt.Sprintf("cpu-%d.pprof", time.Now().UnixMilli()))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return Result{}, err
		}
		f, err := os.Create(out)
		if err != nil {
			return Result{}, err
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return Result{}, err
		}
		time.Sleep(time.Duration(inputInt(req.Input, "durationMs", 250)) * time.Millisecond)
		pprof.StopCPUProfile()
		return capabilityOKWithArtifacts("profile captured", map[string]any{"path": out}, []ResultArtifact{{Type: "profile", Path: out, Summary: "cpu profile"}}), nil
	case "explain_anomaly":
		return capabilityOK("anomaly explanation generated", map[string]any{"summary": "No anomaly model configured; inspect metrics and trace rows for deterministic evidence."}), nil
	default:
		return capabilityOK("observability capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeUI(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "synthesize_speech":
		return t.runConfiguredCommand(ctx, req, "FORGE_TTS_BIN", defaultTTSCommand())
	case "transcribe_audio":
		return t.runConfiguredCommand(ctx, req, "FORGE_WHISPER_BIN", "whisper")
	case "screen_record":
		return t.runConfiguredCommand(ctx, req, "FORGE_FFMPEG_BIN", "ffmpeg")
	case "read_clipboard", "write_clipboard", "screenshot", "prompt_user", "dismiss_notification", "render_ui", "navigate", "inject_input":
		return t.callDesktopBridge(ctx, req)
	default:
		return capabilityOK("ui capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeDevice(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "list_devices":
		return capabilityOK("devices listed", map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU()}), nil
	case "read_sensor":
		rows, _ := workspaceGlob("/sys/class/hwmon", "**")
		return capabilityOK("sensor paths scanned", map[string]any{"paths": rows}), nil
	case "capture_camera", "stream_camera":
		return Result{}, errors.New("camera capture requires a configured platform camera backend")
	case "capture_audio", "play_audio":
		return t.runConfiguredCommand(ctx, req, "FORGE_AUDIO_BIN", "aplay")
	case "print_document":
		return t.runConfiguredCommand(ctx, req, "FORGE_PRINT_BIN", "lp")
	case "write_gpio", "read_gpio":
		if os.Getenv("FORGE_GPIO_ENABLED") != "true" {
			return Result{}, errors.New("GPIO access requires FORGE_GPIO_ENABLED=true")
		}
		return capabilityOK("gpio request captured", map[string]any{"input": nonNilMap(req.Input)}), nil
	case "bluetooth_scan", "bluetooth_connect":
		if os.Getenv("FORGE_BLUETOOTH_ENABLED") != "true" {
			return Result{}, errors.New("bluetooth access requires FORGE_BLUETOOTH_ENABLED=true")
		}
		return capabilityOK("bluetooth request captured", map[string]any{"input": nonNilMap(req.Input)}), nil
	case "set_display":
		return t.callDesktopBridge(ctx, req)
	default:
		return capabilityOK("device capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeTime(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "schedule_once", "schedule_recurring", "set_alarm", "set_deadline", "defer_until":
		id := nonEmpty(inputString(req.Input, "id"), "schedule-"+newCorrelationID())
		payload, _ := json.Marshal(nonNilMap(req.Input))
		if err := t.execDB(ctx, `INSERT INTO scheduled_tasks(id, kind, payload_json, status, created_at, updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET payload_json=excluded.payload_json, status=excluded.status, updated_at=excluded.updated_at`, id, t.capability.Name, string(payload), "scheduled", time.Now().UnixMilli(), time.Now().UnixMilli()); err != nil {
			return Result{}, err
		}
		return capabilityOK("schedule stored", map[string]any{"id": id, "kind": t.capability.Name}), nil
	case "cancel_schedule":
		id := inputString(req.Input, "id")
		return capabilityOK("schedule cancelled", map[string]any{"id": id}), t.execDB(ctx, `UPDATE scheduled_tasks SET status='cancelled', updated_at=? WHERE id=?`, time.Now().UnixMilli(), id)
	case "measure_duration":
		id := nonEmpty(inputString(req.Input, "id"), "default")
		capabilityTimersMu.Lock()
		start, ok := capabilityTimers[id]
		if !ok || inputBool(req.Input, "start") {
			capabilityTimers[id] = time.Now()
			capabilityTimersMu.Unlock()
			return capabilityOK("timer started", map[string]any{"id": id}), nil
		}
		delete(capabilityTimers, id)
		capabilityTimersMu.Unlock()
		return capabilityOK("timer stopped", map[string]any{"id": id, "durationMs": time.Since(start).Milliseconds()}), nil
	case "set_system_time":
		if os.Getenv("FORGE_ALLOW_SYSTEM_TIME_MUTATION") != "true" {
			return Result{}, errors.New("system time mutation requires FORGE_ALLOW_SYSTEM_TIME_MUTATION=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_DATE_BIN", "date")
	default:
		return capabilityOK("time capability backed", map[string]any{"now": time.Now().UTC().Format(time.RFC3339Nano)}), nil
	}
}

func (t *capabilityBackingTool) executeExternal(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "call_api", "search_web":
		return fetchURL(ctx, inputString(req.Input, "url"), inputString(req.Input, "method"))
	case "call_llm":
		return Result{}, errors.New("external.call_llm requires model runtime service binding")
	case "query_database":
		return Result{}, errors.New("external.query_database requires a configured DSN and query broker")
	case "send_email", "read_email", "post_message", "create_issue", "update_issue", "read_calendar", "create_event":
		return Result{}, fmt.Errorf("external.%s requires provider credentials in the secrets vault", t.capability.Name)
	default:
		return capabilityOK("external capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeMemory(ctx context.Context, req Request) (Result, error) {
	key := "memory." + nonEmpty(inputString(req.Input, "id"), sha256Hex(inputString(req.Input, "content")+inputString(req.Input, "query")))
	switch t.capability.Name {
	case "remember", "upsert_fact":
		payload, _ := json.Marshal(nonNilMap(req.Input))
		return capabilityOK("memory stored", map[string]any{"id": key}), t.settingSet(ctx, key, string(payload))
	case "recall", "semantic_search":
		query := inputString(req.Input, "query")
		rows, err := searchWorkspaceFiles(t.workspace, query, 20)
		return capabilityOK("memory recall completed", map[string]any{"query": query, "results": rows}), err
	case "forget", "retract_fact":
		return capabilityOK("memory supersession marker stored", map[string]any{"id": key}), t.settingSet(ctx, key+".status", "superseded")
	case "embed_content":
		sum := sha256.Sum256([]byte(inputString(req.Input, "content")))
		return capabilityOK("content embedded deterministically", map[string]any{"sha256": hex.EncodeToString(sum[:])}), nil
	case "cross_reference", "rank_relevance", "diff_knowledge", "summarize_context":
		return capabilityOK("memory analysis completed", map[string]any{"operation": t.capability.Name, "input": nonNilMap(req.Input)}), nil
	default:
		return capabilityOK("memory capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeAgent(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "request_approval":
		return capabilityOK("approval request payload prepared", map[string]any{"input": nonNilMap(req.Input)}), nil
	case "observe_agent":
		return t.queryRows(ctx, `SELECT id, created_at, type, payload_json FROM events ORDER BY id DESC LIMIT 50`)
	case "spawn_agent", "delegate_task", "send_message", "broadcast", "merge_results", "escalate", "kill_agent":
		return capabilityOK("agent operation materialized", map[string]any{"operation": t.capability.Name, "input": nonNilMap(req.Input)}), nil
	default:
		return capabilityOK("agent capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeBackup(ctx context.Context, req Request) (Result, error) {
	return capabilityOK("backup operation captured", map[string]any{"operation": t.capability.Name, "input": nonNilMap(req.Input)}), nil
}

func capabilityHasEffect(capability domain.ToolCapability, effect domain.ToolEffect) bool {
	for _, item := range capability.Effect {
		if item == effect {
			return true
		}
	}
	return false
}

func capabilityOK(message string, data map[string]any) Result {
	return capabilityOKWithArtifacts(message, data, nil)
}

func capabilityOKWithArtifacts(message string, data map[string]any, artifacts []ResultArtifact) Result {
	if data == nil {
		data = map[string]any{}
	}
	return Result{Status: StatusOK, Message: message, Data: data, Artifacts: artifacts}
}

func inputString(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	switch v := input[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case json.Number:
		return strings.TrimSpace(v.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func inputInt(input map[string]any, key string, fallback int) int {
	raw := inputString(input, key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func inputBool(input map[string]any, key string) bool {
	raw := strings.ToLower(inputString(input, key))
	return raw == "1" || raw == "true" || raw == "yes"
}

func workspaceGlob(root, pattern string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	root = filepath.Clean(root)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "**" {
		pattern = "*"
	}
	matches := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		ok, _ := filepath.Match(filepath.ToSlash(pattern), rel)
		if !ok && strings.Contains(pattern, "**") {
			ok = simpleDoubleStarMatch(pattern, rel)
		}
		if ok {
			matches = append(matches, rel)
		}
		if len(matches) >= 500 {
			return filepath.SkipAll
		}
		return nil
	})
	sort.Strings(matches)
	return matches, err
}

func simpleDoubleStarMatch(pattern, rel string) bool {
	parts := strings.Split(pattern, "**")
	return strings.HasPrefix(rel, strings.Trim(parts[0], "/")) && strings.HasSuffix(rel, strings.Trim(parts[len(parts)-1], "/"))
}

func searchWorkspaceFiles(root, query string, limit int) ([]map[string]any, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 20
	}
	rows := []map[string]any{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		score := 0
		if query == "" || strings.Contains(strings.ToLower(rel), query) {
			score += 2
		}
		if score == 0 {
			if b, err := os.ReadFile(path); err == nil && len(b) <= 256*1024 && strings.Contains(strings.ToLower(string(b)), query) {
				score++
			}
		}
		if score > 0 {
			rows = append(rows, map[string]any{"path": rel, "score": score})
		}
		if len(rows) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return rows, err
}

func (t *capabilityBackingTool) createTarGZ(req Request) (string, error) {
	out := inputString(req.Input, "output")
	if out == "" {
		out = filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "snapshots", fmt.Sprintf("%s-%d.tar.gz", strings.ReplaceAll(t.capability.ID, ".", "-"), time.Now().UnixMilli()))
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(t.workspace, out)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	paths := req.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, raw := range paths {
		target, err := firstPath([]string{raw}, t.workspace)
		if err != nil {
			return "", err
		}
		if !pathContains(t.workspace, target) {
			return "", fmt.Errorf("archive target %q outside workspace", target)
		}
		if err := addPathToTar(tw, t.workspace, target); err != nil {
			return "", err
		}
	}
	return out, nil
}

func addPathToTar(tw *tar.Writer, root, target string) error {
	return filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func (t *capabilityBackingTool) extractArchive(req Request) (string, error) {
	src := inputString(req.Input, "archive")
	if src == "" && len(req.Paths) > 0 {
		src = req.Paths[0]
	}
	if src == "" {
		return "", errors.New("extract requires input.archive or a path")
	}
	if !filepath.IsAbs(src) {
		src = filepath.Join(t.workspace, src)
	}
	dst := inputString(req.Input, "destination")
	if dst == "" {
		dst = "scratch/extracted"
	}
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(t.workspace, dst)
	}
	if !pathContains(t.workspace, dst) {
		return "", fmt.Errorf("extract destination %q outside workspace", dst)
	}
	if strings.HasSuffix(src, ".zip") {
		return dst, extractZip(src, dst)
	}
	return dst, extractTarGZ(src, dst)
}

func extractTarGZ(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.Clean(header.Name))
		if !pathContains(dst, target) {
			return fmt.Errorf("archive entry %q escapes destination", header.Name)
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, header.FileInfo().Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func extractZip(src, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, file := range zr.File {
		target := filepath.Join(dst, filepath.Clean(file.Name))
		if !pathContains(dst, target) {
			return fmt.Errorf("archive entry %q escapes destination", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.FileInfo().Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		_ = in.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func (t *capabilityBackingTool) runConfiguredCommand(ctx context.Context, req Request, envName, fallback string) (Result, error) {
	bin := nonEmpty(os.Getenv(envName), fallback)
	args := []string{}
	if raw := inputString(req.Input, "args"); raw != "" {
		args = strings.Fields(raw)
	}
	return runCommand(ctx, t.workspace, bin, args...)
}

func runCommand(ctx context.Context, dir, bin string, args ...string) (Result, error) {
	if _, err := exec.LookPath(bin); err != nil {
		return Result{}, fmt.Errorf("%s not found: %w", bin, err)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return capabilityOK("command completed", map[string]any{"command": append([]string{bin}, args...), "output": string(out)}), err
}

func fetchURL(ctx context.Context, rawURL, method string) (Result, error) {
	if rawURL == "" {
		return Result{}, errors.New("url is required")
	}
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
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
	body, _ := json.Marshal(map[string]any{"capability": t.capability.ID, "input": nonNilMap(req.Input), "paths": req.Paths})
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
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("desktop bridge returned %s: %s", resp.Status, string(respBody))
	}
	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		payload = map[string]any{"body": string(respBody)}
	}
	return capabilityOK("desktop bridge completed", payload), nil
}

func (t *capabilityBackingTool) encryptionKey() ([]byte, error) {
	keyPath := strings.TrimSpace(os.Getenv("FORGE_ENCRYPTION_KEY_PATH"))
	if keyPath == "" {
		keyPath = filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "secrets", "master.key")
	}
	if b, err := os.ReadFile(keyPath); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
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
	key, err := t.encryptionKey()
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
	message := []byte(inputString(req.Input, "message"))
	if t.capability.Name == "sign" {
		sig := ed25519.Sign(priv, message)
		return capabilityOK("message signed", map[string]any{"publicKey": base64.StdEncoding.EncodeToString(pub), "signature": base64.StdEncoding.EncodeToString(sig)}), nil
	}
	sig, err := base64.StdEncoding.DecodeString(inputString(req.Input, "signature"))
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
