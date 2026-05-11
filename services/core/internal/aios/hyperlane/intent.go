package hyperlane

// IntentType identifies a deterministic Hyperlane routing classification.
type IntentType string

const (
	IntentStatusQuery           IntentType = "status_query"
	IntentDiagnosticsQuery      IntentType = "diagnostics_query"
	IntentChatMemoryInspection  IntentType = "chat_memory_inspection"
	IntentChatHistoryLookup     IntentType = "chat_history_lookup"
	IntentRestoreInspection     IntentType = "restore_inspection"
	IntentDreamReportInspection IntentType = "dream_report_inspection"
	IntentMkdir                 IntentType = "mkdir"
	IntentReadFile              IntentType = "read_file"
	IntentWriteFile             IntentType = "write_file"
	IntentListDirectory         IntentType = "list_directory"
	IntentRunCommand            IntentType = "run_command"
	IntentGenerateTemplate      IntentType = "generate_template"
	IntentGatewayToolRequest    IntentType = "gateway_tool_request"
	IntentModelruntimeStatus    IntentType = "modelruntime_status"
	IntentModelRuntimeStatus    IntentType = IntentModelruntimeStatus
	IntentUnknown               IntentType = "unknown"
)

const ParserVersion = "hyperlane.intent.v0.1"

const (
	RouteStructuredStatus      = "structured.status"
	RouteStructuredDiagnostics = "structured.diagnostics"
	RouteChatMemoryInspector   = "structured.chat_memory"
	RouteChatHistoryLookup     = "structured.chat_history_lookup"
	RouteRestoreInspector      = "structured.restore_inspector"
	RouteDreamReportInspector  = "structured.dream_reports"
	RouteModelruntimeStatus    = "structured.modelruntime_status"
	RouteGatewayFSMkdir        = "fs.mkdir"
	RouteGatewayFSWrite        = "fs.write"
	RouteGatewayFSRead         = "fs.read"
	RouteGatewayFSList         = "fs.list"
	RouteGatewayProcRun        = "proc.run"
	RouteGatewayWebSearch      = "web.search"
	RouteGatewayNetFetch       = "net.fetch"
	RouteGatewayDesktopOpen    = "desktop.open"
	RouteGatewayGitStatus      = "git.status"
	RouteModelRuntime          = "modelruntime"
	RouteUnknown               = "unknown"
	RouteStatusQuery           = RouteStructuredStatus
	RouteDiagnosticsQuery      = RouteStructuredDiagnostics
	RouteChatMemoryInspection  = RouteChatMemoryInspector
	RouteChatHistoryLookupHint = RouteChatHistoryLookup
	RouteRestoreInspection     = RouteRestoreInspector
	RouteDreamReportInspection = RouteDreamReportInspector
	RouteModelRuntimeStatus    = RouteModelruntimeStatus
)

// Intent is a CPU-only route proposal. It cannot execute tools, mutate state,
// call modelruntime, or bypass gateway/kernel authority.
type Intent struct {
	ID                   string         `json:"id"`
	Type                 IntentType     `json:"type"`
	Confidence           float64        `json:"confidence"`
	Lane                 string         `json:"lane"`
	Route                string         `json:"route"`
	RequiresGateway      bool           `json:"requires_gateway"`
	RequiresModel        bool           `json:"requires_model"`
	RequiresApprovalHint bool           `json:"requires_approval_hint"`
	RiskClass            string         `json:"risk_class"`
	Arguments            map[string]any `json:"arguments,omitempty"`
	Warnings             []string       `json:"warnings,omitempty"`
	MatchedRule          string         `json:"matched_rule,omitempty"`
	Trace                IntentTrace    `json:"trace"`
}

type IntentTrace struct {
	ParserVersion  string   `json:"parser_version"`
	MatchedRule    string   `json:"matched_rule,omitempty"`
	Confidence     float64  `json:"confidence"`
	Route          string   `json:"route"`
	Warnings       []string `json:"warnings,omitempty"`
	RejectedReason string   `json:"rejected_reason,omitempty"`
}

func UnknownIntent(id, rejectedReason string, warnings []string) Intent {
	return Intent{
		ID:            id,
		Type:          IntentUnknown,
		Confidence:    0,
		Lane:          "operator",
		Route:         RouteUnknown,
		RequiresModel: true,
		RiskClass:     "none",
		Warnings:      append([]string(nil), warnings...),
		MatchedRule:   "unknown",
		Trace: IntentTrace{
			ParserVersion:  ParserVersion,
			MatchedRule:    "unknown",
			Confidence:     0,
			Route:          RouteUnknown,
			Warnings:       append([]string(nil), warnings...),
			RejectedReason: rejectedReason,
		},
	}
}

func SupportsNoModelRoute(intent Intent) bool {
	if intent.RequiresModel || intent.RequiresGateway {
		return false
	}
	switch intent.Route {
	case RouteStructuredStatus, RouteStructuredDiagnostics, RouteChatMemoryInspector, RouteChatHistoryLookup, RouteRestoreInspector, RouteDreamReportInspector, RouteModelruntimeStatus:
		return true
	default:
		return false
	}
}
