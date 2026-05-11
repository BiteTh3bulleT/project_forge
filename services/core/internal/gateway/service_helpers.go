package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/lanes"
)

// --- helpers ---

func laneCovers(lane *lanes.Lane, target string) bool {
	if len(lane.AllowedPaths) == 0 {
		return false
	}
	for _, s := range lane.ForbiddenPaths {
		if pathContains(s, target) {
			return false
		}
	}
	for _, a := range lane.AllowedPaths {
		if pathContains(a, target) {
			return true
		}
	}
	return false
}

func pathContains(scope, target string) bool {
	if scope == "" || target == "" {
		return false
	}
	scope = expandUserPath(scope)
	target = expandUserPath(target)
	absScope, err := filepath.Abs(scope)
	if err != nil {
		absScope = scope
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		absTarget = target
	}
	absScope = filepath.Clean(absScope)
	absTarget = filepath.Clean(absTarget)
	if absTarget == absScope {
		return true
	}
	rel, err := filepath.Rel(absScope, absTarget)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}

func resolvePaths(workspace string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = expandUserPath(p)
		if !filepath.IsAbs(p) {
			p = filepath.Join(workspace, p)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		out = append(out, filepath.Clean(abs))
	}
	return out
}

func firstPath(paths []string, workspace string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("this tool requires at least one path")
	}
	p := expandUserPath(strings.TrimSpace(paths[0]))
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func firstWorkspacePath(paths []string, workspace string) (string, error) {
	target, err := firstPath(paths, workspace)
	if err != nil {
		return "", err
	}
	if err := validateWorkspacePath(workspace, target); err != nil {
		return "", err
	}
	return target, nil
}

func workspaceDirFromRequest(paths []string, workspace string) (string, error) {
	if len(paths) == 0 {
		return workspace, nil
	}
	return firstWorkspacePath(paths, workspace)
}

func validateWorkspacePath(workspace, target string) error {
	if strings.TrimSpace(workspace) == "" {
		return errors.New("workspace path is required")
	}
	if !pathContains(workspace, target) {
		return fmt.Errorf("path %q is outside workspace", target)
	}
	resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return fmt.Errorf("resolve workspace %q: %w", workspace, err)
	}
	existing, err := nearestExistingPath(target)
	if err != nil {
		return err
	}
	resolvedExisting, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", existing, err)
	}
	if !pathContains(resolvedWorkspace, resolvedExisting) {
		return fmt.Errorf("path %q escapes workspace through symlink path", target)
	}
	return nil
}

func expandUserPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~/"))
}

func writeBytesFromInput(input map[string]any) int64 {
	if input == nil {
		return 0
	}
	if v, ok := input["contents"].(string); ok {
		return int64(len(v))
	}
	if rawFiles, ok := input["files"]; ok {
		files, err := writeBatchFiles(rawFiles)
		if err == nil {
			var total int64
			for _, file := range files {
				total += int64(len(file.Contents))
			}
			return total
		}
	}
	if v, ok := input["bytes"].(float64); ok {
		return int64(v)
	}
	return 0
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func mergeGatewayMetadata(base map[string]any, add map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range add {
		if strings.TrimSpace(fmt.Sprintf("%v", value)) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func gatewayAuditContextPayload(req Request, payload map[string]any) map[string]any {
	out := map[string]any{
		"correlationId": req.CorrelationID,
	}
	if strings.TrimSpace(req.TraceID) != "" {
		out["traceId"] = req.TraceID
	}
	if strings.TrimSpace(req.WorkspaceID) != "" {
		out["workspaceId"] = req.WorkspaceID
	}
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func summarizeResultArtifacts(items []ResultArtifact) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := map[string]any{
			"type":    strings.TrimSpace(item.Type),
			"path":    strings.TrimSpace(item.Path),
			"summary": strings.TrimSpace(item.Summary),
		}
		out = append(out, entry)
	}
	return out
}

func capabilityIDFromRequest(req Request) string {
	if req.Metadata == nil {
		return ""
	}
	return metadataString(req.Metadata, "toolCapabilityId")
}

func metadataBool(meta map[string]any, key string) bool {
	if meta == nil {
		return false
	}
	v, ok := meta[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		x = strings.TrimSpace(strings.ToLower(x))
		return x == "true" || x == "1" || x == "yes"
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

func workspaceIDFromPath(path string) string {
	base := strings.TrimSpace(filepath.Base(filepath.Clean(path)))
	if base == "" || base == "." {
		return "workspace:default"
	}
	base = strings.ToLower(strings.ReplaceAll(base, " ", "_"))
	return "workspace:" + base
}

func mapGatewayStatusToToolStatus(status string) domain.ToolResultStatus {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case StatusOK:
		return domain.ToolStatusSucceeded
	case StatusDryRun:
		return domain.ToolStatusDryRun
	case StatusNeedsApprov:
		return domain.ToolStatusApprovalRequired
	case StatusUnsupported:
		return domain.ToolStatusUnsupported
	case StatusDisabled:
		return domain.ToolStatusDisabled
	case StatusDenied:
		return domain.ToolStatusDenied
	default:
		return domain.ToolStatusFailed
	}
}

func cloneToolOutput(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func marshalGatewayInvocationInput(input map[string]any, metadata map[string]any) ([]byte, error) {
	payload := cloneToolOutput(nonNilMap(input))
	if len(metadata) > 0 {
		payload["_metadata"] = cloneToolOutput(metadata)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(body) <= maxGatewayInvocationInputJSONBytes {
		return body, nil
	}
	summary := map[string]any{
		"_inputOmitted": true,
		"_reason":       "gateway invocation input exceeded persisted JSON limit",
		"_inputSummary": capabilityInputSummary(input),
	}
	if len(metadata) > 0 {
		summary["_metadataSummary"] = capabilityInputSummary(metadata)
	}
	body, err = json.Marshal(summary)
	if err != nil {
		return nil, err
	}
	if len(body) > maxGatewayInvocationInputJSONBytes {
		return nil, fmt.Errorf("gateway invocation input summary too large: %d > %d bytes", len(body), maxGatewayInvocationInputJSONBytes)
	}
	return body, nil
}

func toolErrorFromGatewayResult(result *Result) *domain.ToolExecutionError {
	if result == nil {
		return &domain.ToolExecutionError{Code: domain.ToolErrExecutionFailed, Message: "gateway result is nil"}
	}
	msg := strings.TrimSpace(result.DeniedReason)
	if msg == "" && strings.TrimSpace(result.Message) != "" && strings.TrimSpace(strings.ToLower(result.Status)) != StatusOK {
		msg = strings.TrimSpace(result.Message)
	}
	if msg == "" {
		return nil
	}
	code := domain.ToolErrPolicyDenied
	switch strings.TrimSpace(strings.ToLower(result.Status)) {
	case StatusNeedsApprov:
		code = domain.ToolErrApprovalRequired
	case StatusUnsupported:
		code = domain.ToolErrUnsupportedOperation
	case StatusDisabled:
		code = domain.ToolErrToolDisabled
	case StatusError:
		code = domain.ToolErrExecutionFailed
	}
	return &domain.ToolExecutionError{
		Code:    code,
		Message: msg,
	}
}

func warningsFromGatewayResult(result *Result) []string {
	if result == nil {
		return nil
	}
	raw, ok := result.Data["warnings"]
	if !ok {
		return nil
	}
	out := []string{}
	switch rows := raw.(type) {
	case []string:
		return rows
	case []any:
		for _, item := range rows {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func appendStringWarning(existing any, warning string) []string {
	out := []string{}
	switch rows := existing.(type) {
	case []string:
		out = append(out, rows...)
	case []any:
		for _, row := range rows {
			text := strings.TrimSpace(fmt.Sprintf("%v", row))
			if text != "" {
				out = append(out, text)
			}
		}
	}
	warning = strings.TrimSpace(warning)
	if warning != "" {
		out = append(out, warning)
	}
	return out
}

func newCorrelationID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "corr-" + hex.EncodeToString(buf[:])
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func (g *Gateway) approvalGrantPresent(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (bool, error) {
	if strings.TrimSpace(req.ApprovalID) != "" && g.approvals != nil {
		requestID, err := strconv.ParseInt(strings.TrimSpace(req.ApprovalID), 10, 64)
		if err != nil {
			return false, fmt.Errorf("invalid approval id %q", req.ApprovalID)
		}
		approvalReq, err := g.approvals.GetRequest(ctx, requestID)
		if err != nil {
			return false, err
		}
		if req.JobID != nil && strings.TrimSpace(*req.JobID) != "" && strings.TrimSpace(approvalReq.JobID) != strings.TrimSpace(*req.JobID) {
			return false, fmt.Errorf("approval request %d belongs to job %q, not %q", requestID, approvalReq.JobID, strings.TrimSpace(*req.JobID))
		}
		reqForFingerprint := req
		if reqForFingerprint.JobID == nil || strings.TrimSpace(*reqForFingerprint.JobID) == "" {
			jid := strings.TrimSpace(approvalReq.JobID)
			reqForFingerprint.JobID = &jid
		}
		actualHash, _ := g.approvalFingerprintForRequestID(reqForFingerprint, lane, tool, risk, level, resolvedPaths, requestID)
		expectedHash := approvalFingerprintHashFromScope(approvalReq.ScopeSnapshot)
		if expectedHash == "" {
			return false, fmt.Errorf("approval request %d is missing gateway approval fingerprint", requestID)
		}
		if actualHash != expectedHash {
			return false, fmt.Errorf("approval request %d fingerprint mismatch", requestID)
		}
		if approvalReq.Decision != nil && strings.EqualFold(strings.TrimSpace(approvalReq.Decision.Decision), "approved") {
			return true, nil
		}
		return false, nil
	}
	return g.jobApprovalGranted(ctx, req.JobID)
}

func (g *Gateway) approvalFingerprint(req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (string, map[string]any) {
	return g.approvalFingerprintForRequestID(req, lane, tool, risk, level, resolvedPaths, 0)
}

func (g *Gateway) approvalFingerprintForRequestID(req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string, approvalRequestID int64) (string, map[string]any) {
	jobID := ""
	if req.JobID != nil {
		jobID = strings.TrimSpace(*req.JobID)
	}
	fields := map[string]any{
		"version":          "gateway.v1",
		"actorId":          nonEmpty(req.ProvenanceActor, req.Initiator),
		"actorKind":        nonEmpty(req.ProvenanceActorType, req.Source),
		"initiator":        nonEmpty(req.Initiator, "operator"),
		"source":           nonEmpty(req.Source, "user"),
		"workspaceId":      nonEmpty(req.WorkspaceID, workspaceIDFromPath(g.workspace)),
		"laneId":           lane.ID,
		"toolId":           tool.ID(),
		"capabilityId":     capabilityIDFromRequest(req),
		"riskClass":        strings.TrimSpace(risk),
		"executionLevel":   strings.TrimSpace(level),
		"writeIntent":      tool.WriteIntent(),
		"jobId":            jobID,
		"domain":           nonEmpty(req.Domain, tool.Domain()),
		"action":           nonEmpty(req.Action, tool.Action()),
		"requestedPaths":   normalizedApprovalPaths(req.Paths),
		"resolvedPaths":    normalizedApprovalPaths(resolvedPaths),
		"inputActionShape": normalizeApprovalFingerprintValue(nonNilMap(req.Input)),
	}
	if approvalRequestID > 0 {
		fields["approvalRequestId"] = approvalRequestID
	}
	body, _ := json.Marshal(fields)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), fields
}

func (g *Gateway) updateApprovalRequestScopeSnapshot(ctx context.Context, requestID int64, scope map[string]any) error {
	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return err
	}
	_, err = g.db.ExecContext(ctx, `UPDATE approval_requests SET scope_snapshot_json = ? WHERE id = ?`, string(scopeJSON), requestID)
	return err
}

func approvalFingerprintHashFromScope(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var scope map[string]any
	if err := json.Unmarshal(raw, &scope); err != nil {
		return ""
	}
	if v, ok := scope["approvalFingerprintHash"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func normalizedApprovalPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "." || p == "" {
			continue
		}
		key := strings.ToLower(filepath.ToSlash(p))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, filepath.ToSlash(p))
	}
	sort.Strings(out)
	return out
}

func normalizeApprovalFingerprintValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, minInt(len(keys), maxApprovalFingerprintCollectionItems)+2)
		truncatedFields := false
		truncatedFieldNames := false
		for i, key := range keys {
			if i >= maxApprovalFingerprintCollectionItems {
				truncatedFields = true
				break
			}
			normalizedKey, truncatedName := normalizeApprovalFingerprintFieldName(key)
			if truncatedName {
				truncatedFieldNames = true
			}
			out[normalizedKey] = normalizeApprovalFingerprintValue(typed[key])
		}
		if truncatedFields {
			out["_truncated"] = true
		}
		if truncatedFieldNames {
			out["_truncatedFieldNames"] = true
		}
		return out
	case []any:
		limit := minInt(len(typed), maxApprovalFingerprintCollectionItems)
		out := make([]any, 0, limit)
		for _, item := range typed[:limit] {
			out = append(out, normalizeApprovalFingerprintValue(item))
		}
		if len(typed) > maxApprovalFingerprintCollectionItems {
			return map[string]any{"items": out, "count": len(typed), "truncated": true}
		}
		return out
	case []string:
		out := append([]string(nil), typed...)
		sort.Strings(out)
		if len(out) > maxApprovalFingerprintCollectionItems {
			items := make([]any, 0, maxApprovalFingerprintCollectionItems)
			for _, item := range out[:maxApprovalFingerprintCollectionItems] {
				items = append(items, normalizeApprovalFingerprintValue(item))
			}
			return map[string]any{"items": items, "count": len(out), "truncated": true}
		}
		normalized := make([]any, 0, len(out))
		changed := false
		for _, item := range out {
			value := normalizeApprovalFingerprintString(item)
			if _, ok := value.(string); !ok {
				changed = true
			}
			normalized = append(normalized, value)
		}
		if changed {
			return normalized
		}
		return out
	case string:
		return normalizeApprovalFingerprintString(typed)
	default:
		return typed
	}
}

func normalizeApprovalFingerprintString(value string) any {
	if len(value) <= maxApprovalFingerprintStringBytes {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return map[string]any{
		"omitted": true,
		"bytes":   len(value),
		"sha256":  "sha256:" + hex.EncodeToString(sum[:]),
	}
}

func normalizeApprovalFingerprintFieldName(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "<empty>", false
	}
	if len(key) <= maxApprovalFingerprintFieldNameBytes {
		return key, false
	}
	sum := sha256.Sum256([]byte(key))
	digest := hex.EncodeToString(sum[:])[:16]
	prefixBytes := maxApprovalFingerprintFieldNameBytes - len(digest) - 1
	if prefixBytes < 1 {
		return digest, true
	}
	return key[:prefixBytes] + "#" + digest, true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (g *Gateway) jobApprovalGranted(ctx context.Context, jobID *string) (bool, error) {
	if !g.jobApprovalStatusGranted(ctx, jobID) {
		return false, nil
	}
	return true, nil
}

func (g *Gateway) jobApprovalStatusGranted(ctx context.Context, jobID *string) bool {
	if jobID == nil {
		return false
	}
	id := strings.TrimSpace(*jobID)
	if id == "" {
		return false
	}
	var status string
	err := g.db.QueryRowContext(ctx, `SELECT approval_status FROM jobs WHERE id = ?`, id).Scan(&status)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(status), "granted")
}

func (g *Gateway) jobApprovalFingerprintGranted(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (bool, error) {
	if req.JobID == nil || strings.TrimSpace(*req.JobID) == "" {
		return false, nil
	}
	jobID := strings.TrimSpace(*req.JobID)
	row := g.db.QueryRowContext(ctx, `
SELECT ar.id, ar.scope_snapshot_json
FROM approval_requests ar
JOIN approval_decisions ad ON ad.request_id = ar.id
WHERE ar.job_id = ? AND ar.status = 'resolved' AND lower(ad.decision) = 'approved'
ORDER BY ad.id DESC
LIMIT 1`, jobID)
	var requestID int64
	var scope string
	if err := row.Scan(&requestID, &scope); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	expectedHash := approvalFingerprintHashFromScope(json.RawMessage(scope))
	if expectedHash == "" {
		return false, fmt.Errorf("approval request %d is missing gateway approval fingerprint", requestID)
	}
	actualHash, _ := g.approvalFingerprintForRequestID(req, lane, tool, risk, level, resolvedPaths, requestID)
	if actualHash != expectedHash {
		if approvalScopeHoldsForRequest(json.RawMessage(scope), req, risk) {
			return true, nil
		}
		return false, fmt.Errorf("approval request %d fingerprint mismatch", requestID)
	}
	return true, nil
}

func approvalScopeHoldsForRequest(raw json.RawMessage, req Request, risk string) bool {
	if len(raw) == 0 {
		return false
	}
	var scope map[string]any
	if err := json.Unmarshal(raw, &scope); err != nil {
		return false
	}
	holdOpen, _ := scope["approvalHoldOpen"].(bool)
	if !holdOpen {
		return false
	}
	approvedCorr := strings.TrimSpace(fmt.Sprintf("%v", scope["approvalHoldCorrelationId"]))
	if approvedCorr != "" && strings.TrimSpace(req.CorrelationID) != "" && approvedCorr != strings.TrimSpace(req.CorrelationID) {
		return false
	}
	approvedJobID := strings.TrimSpace(fmt.Sprintf("%v", scope["approvalHoldJobId"]))
	if approvedJobID != "" {
		if req.JobID == nil || strings.TrimSpace(*req.JobID) != approvedJobID {
			return false
		}
	}
	maxRank := intFromApprovalScope(scope["approvalHoldMaxRiskRank"])
	if maxRank > 0 && gatewayApprovalRiskRank(risk) > maxRank {
		return false
	}
	return true
}

func intFromApprovalScope(v any) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		n, _ := typed.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	default:
		return 0
	}
}

func gatewayApprovalRiskRank(risk string) int {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium", "safe_write":
		return 3
	case "low", "read":
		return 2
	case "none", "":
		return 1
	default:
		return 3
	}
}

func toolDomainFromID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if strings.Contains(id, ".") {
		return strings.Split(id, ".")[0]
	}
	return "unknown"
}

func normalizeExecutionLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "L0":
		return "L0"
	case "L1":
		return "L1"
	case "L2":
		return "L2"
	case "L3":
		return "L3"
	case "L4":
		return "L4"
	default:
		return ""
	}
}

func executionLevelFromRisk(risk string) string {
	switch strings.TrimSpace(strings.ToLower(risk)) {
	case "read_only":
		return "L0"
	case "low_write":
		return "L1"
	case "safe_write":
		return "L1"
	case "scoped_execute":
		return "L2"
	case "privileged":
		return "L3"
	case "dangerous":
		return "L4"
	case "low":
		return "L0"
	case "medium":
		return "L1"
	case "high":
		return "L3"
	default:
		return "L0"
	}
}

func gatewayToolIntrinsicApprovalReason(tool Tool, risk, level string) string {
	if tool == nil {
		return ""
	}
	normalizedRisk := strings.TrimSpace(strings.ToLower(risk))
	if normalizedRisk == "" {
		normalizedRisk = strings.TrimSpace(strings.ToLower(tool.RiskClass()))
	}
	normalizedLevel := normalizeExecutionLevel(level)
	if normalizedLevel == "" {
		normalizedLevel = normalizeExecutionLevel(tool.ExecutionLevel())
	}
	if normalizedLevel == "" {
		normalizedLevel = executionLevelFromRisk(normalizedRisk)
	}
	if levelRank(normalizedLevel) >= levelRank("L3") || normalizedRisk == "privileged" || normalizedRisk == "dangerous" {
		return fmt.Sprintf("tool %q is intrinsically privileged and requires approval", tool.ID())
	}
	return ""
}

func legacyRiskClass(risk string) string {
	switch strings.TrimSpace(strings.ToLower(risk)) {
	case "read_only":
		return "low"
	case "low_write":
		return "low"
	case "safe_write":
		return "medium"
	case "scoped_execute":
		return "medium"
	case "privileged":
		return "high"
	case "dangerous":
		return "high"
	default:
		return strings.TrimSpace(strings.ToLower(risk))
	}
}

func effectiveRiskClass(requested, lane, tool string, extra ...string) string {
	best := strings.TrimSpace(strings.ToLower(tool))
	bestRank := levelRank(executionLevelFromRisk(best))
	candidates := []string{lane, requested}
	candidates = append(candidates, extra...)
	for _, candidate := range candidates {
		risk := strings.TrimSpace(strings.ToLower(candidate))
		if risk == "" {
			continue
		}
		rank := levelRank(executionLevelFromRisk(risk))
		if rank > bestRank {
			best = risk
			bestRank = rank
		}
	}
	if best == "" {
		return "read_only"
	}
	return best
}

func levelRank(level string) int {
	switch normalizeExecutionLevel(level) {
	case "L0":
		return 0
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	case "L4":
		return 4
	default:
		return -1
	}
}

func runCmd(ctx context.Context, dir string, parts ...string) (string, error) {
	if len(parts) == 0 {
		return "", errors.New("command required")
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	return boundedCombinedOutput(cmd)
}

func runDetachedCmd(dir string, parts ...string) (int, error) {
	if len(parts) == 0 {
		return 0, errors.New("command required")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
		_ = cmd.Process.Release()
	}
	return pid, nil
}

func readFloat(in map[string]any, key string, def float64) float64 {
	if in == nil {
		return def
	}
	v, ok := in[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return def
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func maskSecret(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
