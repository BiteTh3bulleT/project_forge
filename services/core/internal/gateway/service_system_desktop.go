package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type serviceStatusTool struct{}

var systemdUnitNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.:@\\-]+$`)

const (
	defaultJournalTailLines = 100
	maxJournalTailLines     = 500
)

func normalizeSystemdUnitName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("systemd unit name required")
	}
	if len(name) > 256 {
		return "", errors.New("systemd unit name too long")
	}
	if strings.HasPrefix(name, "-") {
		return "", errors.New("systemd unit name must not start with '-'")
	}
	if strings.Contains(name, "..") || !systemdUnitNamePattern.MatchString(name) {
		return "", errors.New("systemd unit name contains unsafe characters")
	}
	return name, nil
}

func normalizeJournalTailLines(input map[string]any) int {
	lines := int(readFloat(input, "lines", defaultJournalTailLines))
	if lines <= 0 {
		return defaultJournalTailLines
	}
	if lines > maxJournalTailLines {
		return maxJournalTailLines
	}
	return lines
}

func (t *serviceStatusTool) ID() string             { return "system.service_status" }
func (t *serviceStatusTool) Domain() string         { return "system" }
func (t *serviceStatusTool) Action() string         { return "service_status" }
func (t *serviceStatusTool) RiskClass() string      { return "read_only" }
func (t *serviceStatusTool) ExecutionLevel() string { return "L0" }
func (t *serviceStatusTool) Executes() bool         { return true }
func (t *serviceStatusTool) UsesNetwork() bool      { return false }
func (t *serviceStatusTool) WriteIntent() bool      { return false }
func (t *serviceStatusTool) Description() string    { return "Inspect system service status (systemctl)" }
func (t *serviceStatusTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, err := normalizeSystemdUnitName(inputString(req.Input, "service"))
	if err != nil {
		return Result{}, errors.New("system.service_status requires input.service")
	}
	out, err := runCmd(ctx, "", "systemctl", "status", "--no-pager", "--", name)
	return Result{Data: map[string]any{"service": name, "output": out, "ok": err == nil}, Message: "service status checked"}, nil
}

type serviceControlTool struct{}

func (t *serviceControlTool) ID() string             { return "system.service_control" }
func (t *serviceControlTool) Domain() string         { return "system" }
func (t *serviceControlTool) Action() string         { return "service_control" }
func (t *serviceControlTool) RiskClass() string      { return "privileged" }
func (t *serviceControlTool) ExecutionLevel() string { return "L3" }
func (t *serviceControlTool) Executes() bool         { return true }
func (t *serviceControlTool) UsesNetwork() bool      { return false }
func (t *serviceControlTool) WriteIntent() bool      { return true }
func (t *serviceControlTool) Description() string    { return "Start/stop/restart a service (systemctl)" }
func (t *serviceControlTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, nameErr := normalizeSystemdUnitName(inputString(req.Input, "service"))
	action, _ := req.Input["control"].(string)
	action = strings.TrimSpace(strings.ToLower(action))
	if nameErr != nil || (action != "start" && action != "stop" && action != "restart") {
		return Result{}, errors.New("system.service_control requires input.service and control=start|stop|restart")
	}
	out, err := runCmd(ctx, "", "systemctl", action, "--", name)
	return Result{Data: map[string]any{"service": name, "control": action, "output": out, "ok": err == nil}, Message: "service control executed"}, nil
}

type journalTailTool struct{}

func (t *journalTailTool) ID() string             { return "system.logs" }
func (t *journalTailTool) Domain() string         { return "system" }
func (t *journalTailTool) Action() string         { return "logs" }
func (t *journalTailTool) RiskClass() string      { return "read_only" }
func (t *journalTailTool) ExecutionLevel() string { return "L0" }
func (t *journalTailTool) Executes() bool         { return true }
func (t *journalTailTool) UsesNetwork() bool      { return false }
func (t *journalTailTool) WriteIntent() bool      { return false }
func (t *journalTailTool) Description() string    { return "Tail journal/service logs" }
func (t *journalTailTool) Execute(ctx context.Context, req Request) (Result, error) {
	service := ""
	if strings.TrimSpace(inputString(req.Input, "service")) != "" {
		normalized, err := normalizeSystemdUnitName(inputString(req.Input, "service"))
		if err != nil {
			return Result{}, errors.New("system.logs input.service must be a safe systemd unit name")
		}
		service = normalized
	}
	lines := normalizeJournalTailLines(req.Input)
	args := []string{"-n", strconv.Itoa(lines), "--no-pager"}
	if service != "" {
		args = append(args, "-u", service)
	}
	out, err := runCmd(ctx, "", append([]string{"journalctl"}, args...)...)
	return Result{Data: map[string]any{"service": service, "lines": lines, "output": out, "ok": err == nil}, Message: "logs fetched"}, nil
}

type desktopNotifyTool struct{}

func (t *desktopNotifyTool) ID() string             { return "desktop.notify" }
func (t *desktopNotifyTool) Domain() string         { return "desktop" }
func (t *desktopNotifyTool) Action() string         { return "notify" }
func (t *desktopNotifyTool) RiskClass() string      { return "safe_write" }
func (t *desktopNotifyTool) ExecutionLevel() string { return "L1" }
func (t *desktopNotifyTool) Executes() bool         { return true }
func (t *desktopNotifyTool) UsesNetwork() bool      { return false }
func (t *desktopNotifyTool) WriteIntent() bool      { return true }
func (t *desktopNotifyTool) Description() string {
	return "Send local desktop notification (notify-send)"
}
func (t *desktopNotifyTool) Execute(ctx context.Context, req Request) (Result, error) {
	title, _ := req.Input["title"].(string)
	body, _ := req.Input["body"].(string)
	if strings.TrimSpace(title) == "" {
		title = "FORGE"
	}
	out, err := runCmd(ctx, "", "notify-send", strings.TrimSpace(title), strings.TrimSpace(body))
	return Result{Data: map[string]any{"title": title, "body": body, "output": out, "ok": err == nil}, Message: "notification attempted"}, nil
}

type desktopOpenTool struct{ workspace string }

func (t *desktopOpenTool) ID() string             { return "desktop.open" }
func (t *desktopOpenTool) Domain() string         { return "desktop" }
func (t *desktopOpenTool) Action() string         { return "open_path" }
func (t *desktopOpenTool) RiskClass() string      { return "safe_write" }
func (t *desktopOpenTool) ExecutionLevel() string { return "L1" }
func (t *desktopOpenTool) Executes() bool         { return true }
func (t *desktopOpenTool) UsesNetwork() bool      { return false }
func (t *desktopOpenTool) WriteIntent() bool      { return true }
func (t *desktopOpenTool) Description() string {
	return "Open file/folder/URL or launch desktop app using desktop session"
}
func (t *desktopOpenTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) > 0 {
		target, err := firstWorkspacePath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		pid, out, err := desktopOpenTarget(ctx, target)
		return Result{
			Data:    map[string]any{"mode": "path", "path": target, "pid": pid, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	candidate := desktopInputCandidate(req.Input)
	if candidate == "" {
		return Result{}, errors.New("desktop.open requires either paths[] or input.path|input.url|input.application")
	}
	appHint, inlineCommand := desktopSplitAppAndCommand(candidate)
	if len(inlineCommand) == 0 {
		inlineCommand = desktopInlineCommandFromInput(req.Input)
	}

	if desktopLooksLikeURL(appHint) {
		target, err := validateDesktopOpenURL(appHint)
		if err != nil {
			return Result{}, err
		}
		pid, out, err := desktopOpenTarget(ctx, target)
		return Result{
			Data:    map[string]any{"mode": "url", "target": target, "pid": pid, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	if desktopLooksLikePath(appHint) {
		target, err := firstWorkspacePath([]string{appHint}, t.workspace)
		if err != nil {
			return Result{}, err
		}
		pid, out, err := desktopOpenTarget(ctx, target)
		return Result{
			Data:    map[string]any{"mode": "path", "path": target, "pid": pid, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	command, args, appName, err := desktopResolveAppLaunch(appHint)
	if err != nil {
		return Result{}, err
	}
	if len(inlineCommand) > 0 {
		terminalArgs, ok := desktopTerminalLaunchArgs(command, inlineCommand)
		if !ok {
			return Result{}, fmt.Errorf("application %q does not support inline command execution", appName)
		}
		args = append(args, terminalArgs...)
	}
	pid, runErr := desktopLaunchApp(command, args)
	out := ""
	return Result{
		Data: map[string]any{
			"mode":        "application",
			"application": appName,
			"command":     command,
			"args":        args,
			"inlineCmd":   inlineCommand,
			"pid":         pid,
			"output":      out,
			"ok":          runErr == nil,
		},
		Message: "application launch attempted",
	}, nil
}

func desktopInputCandidate(input map[string]any) string {
	if input == nil {
		return ""
	}
	keys := []string{"path", "url", "uri", "application", "app", "target", "query", "request", "text", "name", "input"}
	for _, key := range keys {
		raw, ok := input[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			s := strings.TrimSpace(value)
			if s != "" {
				return s
			}
		case fmt.Stringer:
			s := strings.TrimSpace(value.String())
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func desktopInlineCommandFromInput(input map[string]any) []string {
	if input == nil {
		return nil
	}
	keys := []string{"query", "request", "text", "input", "target", "application", "app", "name"}
	for _, key := range keys {
		raw, ok := input[key]
		if !ok {
			continue
		}
		text := ""
		switch value := raw.(type) {
		case string:
			text = strings.TrimSpace(value)
		case fmt.Stringer:
			text = strings.TrimSpace(value.String())
		}
		if text == "" {
			continue
		}
		_, cmd := desktopSplitAppAndCommand(text)
		if len(cmd) > 0 {
			return cmd
		}
	}
	return nil
}

func desktopLooksLikeURL(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "mailto:")
}

func validateDesktopOpenURL(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return "", errors.New("desktop.open URL is required")
	}
	if len(normalized) > maxOutboundHTTPURLBytes {
		return "", fmt.Errorf("desktop.open URL too large: %d > %d bytes", len(normalized), maxOutboundHTTPURLBytes)
	}
	if strings.ContainsAny(normalized, "\x00\r\n\t") {
		return "", errors.New("desktop.open URL contains control characters")
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https":
		if parsed.User != nil {
			return "", errors.New("desktop.open URL userinfo is not allowed")
		}
		if strings.TrimSpace(parsed.Hostname()) == "" {
			return "", errors.New("desktop.open URL host is required")
		}
	case "mailto":
		if strings.TrimSpace(parsed.Opaque) == "" && strings.TrimSpace(parsed.Path) == "" {
			return "", errors.New("desktop.open mailto target is required")
		}
	default:
		return "", errors.New("desktop.open only supports http, https, and mailto URLs")
	}
	return parsed.String(), nil
}

func desktopLooksLikePath(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") {
		return true
	}
	return strings.Contains(s, "/")
}

func desktopSplitAppAndCommand(raw string) (appHint string, command []string) {
	if cmd, ok := desktopSSHRemoteMkdirCommand(raw); ok {
		return "terminal", cmd
	}
	if app, cmd, ok := desktopGenericInlineCommand(raw); ok {
		return app, cmd
	}
	normalized := desktopNormalizeAppHint(raw)
	if normalized == "" {
		return "", nil
	}
	if strings.HasPrefix(normalized, "ping ") {
		target := desktopExtractPingTarget(strings.TrimSpace(strings.TrimPrefix(normalized, "ping ")))
		if target != "" {
			return "terminal", []string{"ping", target}
		}
		return normalized, nil
	}
	delims := []string{
		" and run ping ",
		" and ping ",
		" to run ping ",
		" to ping ",
		" then ping ",
	}
	for _, delim := range delims {
		idx := strings.Index(normalized, delim)
		if idx <= 0 {
			continue
		}
		target := desktopExtractPingTarget(strings.TrimSpace(normalized[idx+len(delim):]))
		if target == "" {
			return normalized, nil
		}
		app := strings.TrimSpace(normalized[:idx])
		if app == "" {
			app = "terminal"
		}
		return app, []string{"ping", target}
	}
	return normalized, nil
}

var reDesktopSSHUserHost = regexp.MustCompile(`(?i)\bssh\s+(?:into|to)?\s*([a-z0-9._-]+@[a-z0-9.-]+)\b`)

func desktopSSHRemoteMkdirCommand(raw string) ([]string, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, false
	}
	m := reDesktopSSHUserHost.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil, false
	}
	userHost := strings.TrimSpace(m[1])
	if !desktopSafeSSHUserHost(userHost) {
		return nil, false
	}
	if writePath, contents, ok := ParsePythonBannerScriptIntent(text); ok {
		dir := filepath.ToSlash(filepath.Dir(writePath))
		if dir == "." || dir == "" || hasTraversalSegment(dir) || strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, "~") {
			return nil, false
		}
		remoteScript := fmt.Sprintf("mkdir -p %s && cat > %s <<'PY'\n%s\nPY\npython3 %s", shellQuoteArg(dir), shellQuoteArg(writePath), contents, shellQuoteArg(writePath))
		return []string{"ssh", userHost, remoteScript}, true
	}
	dir, ok := parseDirectoryLabeled(text)
	if !ok {
		dir, ok = ParseDirectoryCalled(text)
	}
	if !ok {
		dir, ok = ParseMkdirShellPath(text)
	}
	if !ok || dir == "" {
		return []string{"ssh", userHost}, true
	}
	return []string{"ssh", userHost, "mkdir", "-p", dir}, true
}

func desktopSafeSSHUserHost(userHost string) bool {
	user, host, ok := strings.Cut(strings.TrimSpace(userHost), "@")
	if !ok || user == "" || host == "" {
		return false
	}
	for _, r := range user {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	for _, r := range host {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '-':
		default:
			return false
		}
	}
	return true
}

func shellQuoteArg(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

func desktopGenericInlineCommand(raw string) (appHint string, command []string, ok bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", nil, false
	}
	lower := strings.ToLower(text)
	delims := []string{
		" and run ",
		" then run ",
		" to run ",
		", run ",
		" and execute ",
		" then execute ",
		" to execute ",
		", execute ",
	}
	for _, delim := range delims {
		idx := strings.Index(lower, delim)
		if idx <= 0 {
			continue
		}
		app := desktopNormalizeAppHint(text[:idx])
		if !desktopHintSupportsInlineCommand(app) {
			continue
		}
		cmd, cmdOK := desktopParseInlineCommand(text[idx+len(delim):])
		if !cmdOK {
			continue
		}
		return app, cmd, true
	}
	return "", nil, false
}

func desktopHintSupportsInlineCommand(appHint string) bool {
	hint := strings.TrimSpace(strings.ToLower(appHint))
	return hint == "terminal" ||
		strings.Contains(hint, "terminal") ||
		strings.Contains(hint, "konsole") ||
		strings.Contains(hint, "xterm") ||
		strings.Contains(hint, "kitty") ||
		strings.Contains(hint, "alacritty")
}

func desktopParseInlineCommand(raw string) ([]string, bool) {
	cmd := strings.TrimSpace(raw)
	cmd = strings.TrimPrefix(cmd, "the command ")
	cmd = strings.TrimSpace(strings.Trim(cmd, " :"))
	cmd = strings.Trim(cmd, `"'`)
	cmd = strings.TrimSpace(strings.TrimRight(cmd, "."))
	if cmd == "" || strings.ContainsAny(cmd, "\x00\r\n") {
		return nil, false
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil, false
	}
	return fields, true
}

func desktopExtractPingTarget(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return ""
	}
	token := strings.Trim(fields[0], `"'`)
	if !desktopSafePingTarget(token) {
		return ""
	}
	return token
}

func desktopSafePingTarget(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.', r == '-', r == ':':
		default:
			return false
		}
	}
	return true
}

func desktopTerminalLaunchArgs(command string, cmd []string) ([]string, bool) {
	if len(cmd) == 0 {
		return nil, false
	}
	switch command {
	case "konsole":
		return append([]string{"--noclose", "-e"}, cmd...), true
	case "x-terminal-emulator", "xfce4-terminal", "alacritty":
		return append([]string{"-e"}, cmd...), true
	case "gnome-terminal":
		return append([]string{"--"}, cmd...), true
	case "kitty":
		return append([]string{}, cmd...), true
	default:
		return nil, false
	}
}

func desktopResolveAppLaunch(raw string) (command string, args []string, appName string, err error) {
	hint := desktopNormalizeAppHint(raw)
	if hint == "" {
		return "", nil, "", errors.New("desktop.open input.application cannot be empty")
	}

	candidates := desktopLaunchCandidates(hint)
	for _, candidate := range candidates {
		if len(candidate) == 0 {
			continue
		}
		if desktopLooksLikeURL(candidate[0]) || strings.HasSuffix(strings.ToLower(candidate[0]), ":") || strings.HasPrefix(strings.ToLower(candidate[0]), "shell:") {
			return candidate[0], candidate[1:], hint, nil
		}
		if _, lookErr := exec.LookPath(candidate[0]); lookErr == nil {
			return candidate[0], candidate[1:], hint, nil
		}
	}
	return "", nil, "", fmt.Errorf("no launcher found for %q on this system", hint)
}

func desktopNormalizeAppHint(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	s = strings.Trim(s, `"'`)
	s = strings.TrimSuffix(s, ".")
	for _, prefix := range []string{"can you ", "could you ", "would you ", "please "} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	for _, prefix := range []string{"open ", "launch ", "start "} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
		if strings.HasPrefix(s, "the "+prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, "the "+prefix))
		}
	}
	s = strings.TrimPrefix(s, "the ")
	s = strings.TrimPrefix(s, "my ")
	s = strings.TrimSuffix(s, " please")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func desktopLaunchCandidates(hint string) [][]string {
	normalized := strings.TrimSpace(strings.ToLower(hint))
	if platform := desktopPlatformLaunchCandidates(normalized); len(platform) > 0 {
		return platform
	}
	switch {
	case normalized == "konsole" || strings.Contains(normalized, "konsole"):
		return [][]string{
			{"konsole"},
			{"gtk-launch", "org.kde.konsole.desktop"},
		}
	case strings.Contains(normalized, "terminal"):
		return [][]string{
			{"x-terminal-emulator"},
			{"konsole"},
			{"gnome-terminal"},
			{"xfce4-terminal"},
			{"kitty"},
			{"alacritty"},
		}
	case strings.Contains(normalized, "software center"),
		strings.Contains(normalized, "software manager"),
		strings.Contains(normalized, "app store"),
		strings.Contains(normalized, "discover"):
		return [][]string{
			{"plasma-discover"},
			{"discover"},
			{"gnome-software"},
			{"snap-store"},
			{"gtk-launch", "org.kde.discover.desktop"},
			{"gtk-launch", "org.gnome.Software.desktop"},
		}
	default:
		fields := strings.Fields(normalized)
		if len(fields) > 0 && desktopSafeCommandToken(fields[0]) {
			return [][]string{{fields[0]}}
		}
	}
	return nil
}

func desktopSafeCommandToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '+':
		default:
			return false
		}
	}
	return true
}
