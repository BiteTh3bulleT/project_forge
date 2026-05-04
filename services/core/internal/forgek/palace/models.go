package palace

import "time"

type RoomRouteStats struct {
	RouteCount    int
	SuccessCount  int
	RejectedCount int
}

type MemoryRoomInput struct {
	RoomID         string
	WorkspaceID    string
	Name           string
	Description    string
	DomainTags     []string
	AnchorRefs     []string
	LinkedRoomRefs []string
	RouteStats     RoomRouteStats
	CreatedAt      time.Time
	Metadata       map[string]any
}

type MemoryRoom struct {
	RoomID         string
	WorkspaceID    string
	Name           string
	Description    string
	DomainTags     []string
	AnchorRefs     []string
	LinkedRoomRefs []string
	RouteStats     RoomRouteStats
	CreatedAt      time.Time
	UpdatedAt      time.Time
	JournalRefs    []string
	Metadata       map[string]any
}

func NewMemoryRoom(input MemoryRoomInput) (MemoryRoom, error) {
	if input.RoomID == "" || input.WorkspaceID == "" || input.Name == "" {
		return MemoryRoom{}, ErrInvalidRoom
	}
	return MemoryRoom{
		RoomID:         input.RoomID,
		WorkspaceID:    input.WorkspaceID,
		Name:           input.Name,
		Description:    input.Description,
		DomainTags:     append([]string(nil), input.DomainTags...),
		AnchorRefs:     append([]string(nil), input.AnchorRefs...),
		LinkedRoomRefs: append([]string(nil), input.LinkedRoomRefs...),
		RouteStats:     input.RouteStats,
		CreatedAt:      input.CreatedAt,
		UpdatedAt:      input.CreatedAt,
		Metadata:       cloneMap(input.Metadata),
	}, nil
}

func (r MemoryRoom) Clone() MemoryRoom {
	r.DomainTags = append([]string(nil), r.DomainTags...)
	r.AnchorRefs = append([]string(nil), r.AnchorRefs...)
	r.LinkedRoomRefs = append([]string(nil), r.LinkedRoomRefs...)
	r.JournalRefs = append([]string(nil), r.JournalRefs...)
	r.Metadata = cloneMap(r.Metadata)
	return r
}

type MemoryAnchorInput struct {
	AnchorID     string
	WorkspaceID  string
	RoomID       string
	Label        string
	ObjectRefs   []string
	Keywords     []string
	Tags         []string
	SourceRefs   []string
	EmbeddingRef string
	CreatedAt    time.Time
	Metadata     map[string]any
}

type MemoryAnchor struct {
	AnchorID     string
	WorkspaceID  string
	RoomID       string
	Label        string
	ObjectRefs   []string
	Keywords     []string
	Tags         []string
	SourceRefs   []string
	EmbeddingRef string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	JournalRefs  []string
	Metadata     map[string]any
}

func NewMemoryAnchor(input MemoryAnchorInput) (MemoryAnchor, error) {
	if input.AnchorID == "" || input.WorkspaceID == "" || input.RoomID == "" || input.Label == "" {
		return MemoryAnchor{}, ErrInvalidAnchor
	}
	if len(input.ObjectRefs) == 0 && len(input.SourceRefs) == 0 {
		return MemoryAnchor{}, ErrInvalidAnchor
	}
	return MemoryAnchor{
		AnchorID:     input.AnchorID,
		WorkspaceID:  input.WorkspaceID,
		RoomID:       input.RoomID,
		Label:        input.Label,
		ObjectRefs:   append([]string(nil), input.ObjectRefs...),
		Keywords:     append([]string(nil), input.Keywords...),
		Tags:         append([]string(nil), input.Tags...),
		SourceRefs:   append([]string(nil), input.SourceRefs...),
		EmbeddingRef: input.EmbeddingRef,
		CreatedAt:    input.CreatedAt,
		UpdatedAt:    input.CreatedAt,
		Metadata:     cloneMap(input.Metadata),
	}, nil
}

func (a MemoryAnchor) Clone() MemoryAnchor {
	a.ObjectRefs = append([]string(nil), a.ObjectRefs...)
	a.Keywords = append([]string(nil), a.Keywords...)
	a.Tags = append([]string(nil), a.Tags...)
	a.SourceRefs = append([]string(nil), a.SourceRefs...)
	a.JournalRefs = append([]string(nil), a.JournalRefs...)
	a.Metadata = cloneMap(a.Metadata)
	return a
}

type CandidateObject struct {
	CandidateID      string
	WorkspaceID      string
	SourceObjectID   string
	SourceType       string
	SourceRefs       []string
	AnchorID         string
	RoomID           string
	RelevanceScore   float64
	RetrievalReason  string
	CandidateSummary string
	CreatedAt        time.Time
	Metadata         map[string]any
}

func (c CandidateObject) Clone() CandidateObject {
	c.SourceRefs = append([]string(nil), c.SourceRefs...)
	c.Metadata = cloneMap(c.Metadata)
	return c
}

func (c CandidateObject) IsExhibit() bool {
	return false
}

func (c CandidateObject) IsAdmittedEvidence() bool {
	return false
}

type RouteResultRecord struct {
	CandidateID  string
	ResultStatus string
	RecordedAt   time.Time
}

type PalaceRouteInput struct {
	RouteID          string
	CaseID           string
	WorkspaceID      string
	QueryText        string
	RouteReason      string
	StartRoomID      string
	VisitedRoomIDs   []string
	AnchorRefs       []string
	CandidateObjects []CandidateObject
	RouteScore       float64
	RouteStrategy    string
	CreatedBy        string
	CreatedAt        time.Time
	ResultRecords    []RouteResultRecord
	JournalRefs      []string
	Metadata         map[string]any
}

type PalaceRoute struct {
	RouteID          string
	CaseID           string
	WorkspaceID      string
	QueryText        string
	RouteReason      string
	StartRoomID      string
	VisitedRoomIDs   []string
	AnchorRefs       []string
	CandidateObjects []CandidateObject
	RouteScore       float64
	RouteStrategy    string
	CreatedBy        string
	CreatedAt        time.Time
	JournalRefs      []string
	ResultRecords    []RouteResultRecord
	Metadata         map[string]any
}

func NewPalaceRoute(input PalaceRouteInput) (PalaceRoute, error) {
	if input.RouteID == "" || input.WorkspaceID == "" || input.StartRoomID == "" || input.CreatedBy == "" {
		return PalaceRoute{}, ErrInvalidRoute
	}
	if input.QueryText == "" && input.RouteReason == "" {
		return PalaceRoute{}, ErrInvalidRoute
	}
	return PalaceRoute{
		RouteID:          input.RouteID,
		CaseID:           input.CaseID,
		WorkspaceID:      input.WorkspaceID,
		QueryText:        input.QueryText,
		RouteReason:      input.RouteReason,
		StartRoomID:      input.StartRoomID,
		VisitedRoomIDs:   append([]string(nil), input.VisitedRoomIDs...),
		AnchorRefs:       append([]string(nil), input.AnchorRefs...),
		CandidateObjects: cloneCandidates(input.CandidateObjects),
		RouteScore:       input.RouteScore,
		RouteStrategy:    input.RouteStrategy,
		CreatedBy:        input.CreatedBy,
		CreatedAt:        input.CreatedAt,
		JournalRefs:      append([]string(nil), input.JournalRefs...),
		ResultRecords:    append([]RouteResultRecord(nil), input.ResultRecords...),
		Metadata:         cloneMap(input.Metadata),
	}, nil
}

func (r PalaceRoute) Clone() PalaceRoute {
	r.VisitedRoomIDs = append([]string(nil), r.VisitedRoomIDs...)
	r.AnchorRefs = append([]string(nil), r.AnchorRefs...)
	r.CandidateObjects = cloneCandidates(r.CandidateObjects)
	r.JournalRefs = append([]string(nil), r.JournalRefs...)
	r.ResultRecords = append([]RouteResultRecord(nil), r.ResultRecords...)
	r.Metadata = cloneMap(r.Metadata)
	return r
}

type RouteQuery struct {
	CaseID               string
	WorkspaceID          string
	QueryText            string
	RouteReason          string
	StartRoomID          string
	Tags                 []string
	ObjectTypePreference string
}

func cloneCandidates(in []CandidateObject) []CandidateObject {
	out := make([]CandidateObject, len(in))
	for i, candidate := range in {
		out[i] = candidate.Clone()
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case []string:
			out[key] = append([]string(nil), typed...)
		default:
			out[key] = value
		}
	}
	return out
}
