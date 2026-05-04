package palace

import (
	"fmt"
	"sync"
	"time"
)

type Service struct {
	mu      sync.RWMutex
	rooms   map[string]MemoryRoom
	anchors map[string]MemoryAnchor
	routes  map[string]PalaceRoute
}

type MemoryPalaceService = Service

func NewService() *Service {
	return &Service{
		rooms:   make(map[string]MemoryRoom),
		anchors: make(map[string]MemoryAnchor),
		routes:  make(map[string]PalaceRoute),
	}
}

func (s *Service) CreateRoom(room MemoryRoom) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.RoomID] = room.Clone()
}

func (s *Service) UpdateRoom(room MemoryRoom) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.RoomID] = room.Clone()
}

func (s *Service) GetRoom(roomID string) (MemoryRoom, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[roomID]
	if !ok {
		return MemoryRoom{}, false
	}
	return room.Clone(), true
}

func (s *Service) LinkRooms(roomID, linkedRoomID string, now time.Time, journalRef string) (MemoryRoom, MemoryRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[roomID]
	if !ok {
		return MemoryRoom{}, MemoryRoom{}, ErrRoomNotFound
	}
	linked, ok := s.rooms[linkedRoomID]
	if !ok || linked.WorkspaceID != room.WorkspaceID {
		return MemoryRoom{}, MemoryRoom{}, ErrRoomNotFound
	}
	room.LinkedRoomRefs = appendUnique(room.LinkedRoomRefs, linkedRoomID)
	linked.LinkedRoomRefs = appendUnique(linked.LinkedRoomRefs, roomID)
	room.UpdatedAt = now
	linked.UpdatedAt = now
	room.JournalRefs = append(room.JournalRefs, journalRef)
	linked.JournalRefs = append(linked.JournalRefs, journalRef)
	s.rooms[roomID] = room.Clone()
	s.rooms[linkedRoomID] = linked.Clone()
	return room.Clone(), linked.Clone(), nil
}

func (s *Service) CreateAnchor(anchor MemoryAnchor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[anchor.RoomID]
	if !ok || room.WorkspaceID != anchor.WorkspaceID {
		return ErrRoomNotFound
	}
	s.anchors[anchor.AnchorID] = anchor.Clone()
	room.AnchorRefs = appendUnique(room.AnchorRefs, anchor.AnchorID)
	room.UpdatedAt = anchor.UpdatedAt
	s.rooms[room.RoomID] = room.Clone()
	return nil
}

func (s *Service) UpdateAnchor(anchor MemoryAnchor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.anchors[anchor.AnchorID]; !ok {
		return ErrAnchorNotFound
	}
	s.anchors[anchor.AnchorID] = anchor.Clone()
	return nil
}

func (s *Service) GetAnchor(anchorID string) (MemoryAnchor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	anchor, ok := s.anchors[anchorID]
	if !ok {
		return MemoryAnchor{}, false
	}
	return anchor.Clone(), true
}

func (s *Service) LinkAnchor(anchorID string, objectRefs, sourceRefs []string, now time.Time, journalRef string) (MemoryAnchor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	anchor, ok := s.anchors[anchorID]
	if !ok {
		return MemoryAnchor{}, ErrAnchorNotFound
	}
	for _, ref := range objectRefs {
		anchor.ObjectRefs = appendUnique(anchor.ObjectRefs, ref)
	}
	for _, ref := range sourceRefs {
		anchor.SourceRefs = appendUnique(anchor.SourceRefs, ref)
	}
	anchor.UpdatedAt = now
	anchor.JournalRefs = append(anchor.JournalRefs, journalRef)
	s.anchors[anchorID] = anchor.Clone()
	return anchor.Clone(), nil
}

func (s *Service) CreateRoute(query RouteQuery, routeID, createdBy string, createdAt time.Time, nextCandidateID func() string) (PalaceRoute, []MemoryRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, ok := s.rooms[query.StartRoomID]
	if !ok || start.WorkspaceID != query.WorkspaceID {
		return PalaceRoute{}, nil, ErrRoomNotFound
	}

	visited := append([]string{start.RoomID}, start.LinkedRoomRefs...)
	visitedSet := make(map[string]bool, len(visited))
	for _, roomID := range visited {
		if room, ok := s.rooms[roomID]; ok && room.WorkspaceID == query.WorkspaceID {
			visitedSet[roomID] = true
		}
	}

	var anchorRefs []string
	var candidates []CandidateObject
	for _, anchor := range s.anchors {
		if anchor.WorkspaceID != query.WorkspaceID || !visitedSet[anchor.RoomID] {
			continue
		}
		roomStats := s.rooms[anchor.RoomID].RouteStats
		score := ScoreAnchor(query, anchor, query.StartRoomID, roomStats)
		if score <= 0 {
			continue
		}
		anchorRefs = appendUnique(anchorRefs, anchor.AnchorID)
		for _, objectRef := range anchor.ObjectRefs {
			candidates = append(candidates, CandidateObject{
				CandidateID:      nextCandidateID(),
				WorkspaceID:      query.WorkspaceID,
				SourceObjectID:   objectRef,
				SourceType:       SourceTypeKernelObject,
				SourceRefs:       append([]string{objectRef}, anchor.SourceRefs...),
				AnchorID:         anchor.AnchorID,
				RoomID:           anchor.RoomID,
				RelevanceScore:   score,
				RetrievalReason:  fmt.Sprintf("deterministic route score %.2f from anchor %s", score, anchor.AnchorID),
				CandidateSummary: anchor.Label,
				CreatedAt:        createdAt,
			})
		}
	}
	SortCandidates(candidates)
	routeScore := 0.0
	if len(candidates) > 0 {
		routeScore = candidates[0].RelevanceScore
	}
	route, err := NewPalaceRoute(PalaceRouteInput{
		RouteID:          routeID,
		CaseID:           query.CaseID,
		WorkspaceID:      query.WorkspaceID,
		QueryText:        query.QueryText,
		RouteReason:      query.RouteReason,
		StartRoomID:      query.StartRoomID,
		VisitedRoomIDs:   visited,
		AnchorRefs:       anchorRefs,
		CandidateObjects: candidates,
		RouteScore:       routeScore,
		RouteStrategy:    RouteStrategyKeywordTagOverlap,
		CreatedBy:        createdBy,
		CreatedAt:        createdAt,
	})
	if err != nil {
		return PalaceRoute{}, nil, err
	}
	s.routes[route.RouteID] = route.Clone()

	updatedRooms := make([]MemoryRoom, 0, len(visitedSet))
	for roomID := range visitedSet {
		room := s.rooms[roomID]
		room.RouteStats.RouteCount++
		room.UpdatedAt = createdAt
		s.rooms[roomID] = room.Clone()
		updatedRooms = append(updatedRooms, room.Clone())
	}
	return route.Clone(), updatedRooms, nil
}

func (s *Service) RecordRouteResult(routeID, candidateID, status string, now time.Time, journalRef string) (PalaceRoute, []MemoryRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[routeID]
	if !ok {
		return PalaceRoute{}, nil, ErrRouteNotFound
	}
	var matched CandidateObject
	for _, candidate := range route.CandidateObjects {
		if candidate.CandidateID == candidateID {
			matched = candidate
			break
		}
	}
	if matched.CandidateID == "" {
		return PalaceRoute{}, nil, ErrInvalidCandidate
	}
	route.ResultRecords = append(route.ResultRecords, RouteResultRecord{CandidateID: candidateID, ResultStatus: status, RecordedAt: now})
	route.JournalRefs = append(route.JournalRefs, journalRef)
	s.routes[routeID] = route.Clone()

	room, ok := s.rooms[matched.RoomID]
	if !ok {
		return route.Clone(), nil, nil
	}
	switch status {
	case RouteResultUseful, RouteResultSubmitted, RouteResultAdmitted:
		room.RouteStats.SuccessCount++
	case RouteResultRejected:
		room.RouteStats.RejectedCount++
	}
	room.UpdatedAt = now
	room.JournalRefs = append(room.JournalRefs, journalRef)
	s.rooms[room.RoomID] = room.Clone()
	return route.Clone(), []MemoryRoom{room.Clone()}, nil
}

func (s *Service) ListRooms(workspaceID string) []MemoryRoom {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MemoryRoom, 0)
	for _, room := range s.rooms {
		if room.WorkspaceID == workspaceID {
			out = append(out, room.Clone())
		}
	}
	return out
}

func (s *Service) ListAnchors(workspaceID string) []MemoryAnchor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MemoryAnchor, 0)
	for _, anchor := range s.anchors {
		if anchor.WorkspaceID == workspaceID {
			out = append(out, anchor.Clone())
		}
	}
	return out
}

func (s *Service) ListRoutes(workspaceID string) []PalaceRoute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PalaceRoute, 0)
	for _, route := range s.routes {
		if route.WorkspaceID == workspaceID {
			out = append(out, route.Clone())
		}
	}
	return out
}

func (s *Service) GetRoute(routeID string) (PalaceRoute, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.routes[routeID]
	if !ok {
		return PalaceRoute{}, false
	}
	return route.Clone(), true
}

func (s *Service) UpdateRoute(route PalaceRoute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[route.RouteID] = route.Clone()
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
