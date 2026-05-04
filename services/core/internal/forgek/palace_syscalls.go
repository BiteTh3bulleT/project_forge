package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/palace"
)

func (k *Kernel) registerPalaceSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallPalaceCreateRoom, Handler: handlePalaceCreateRoom},
		{Name: SyscallPalaceUpdateRoom, Handler: handlePalaceUpdateRoom},
		{Name: SyscallPalaceLinkRooms, Handler: handlePalaceLinkRooms},
		{Name: SyscallPalaceCreateAnchor, Handler: handlePalaceCreateAnchor},
		{Name: SyscallPalaceUpdateAnchor, Handler: handlePalaceUpdateAnchor},
		{Name: SyscallPalaceLinkAnchor, Handler: handlePalaceLinkAnchor},
		{Name: SyscallPalaceRoute, Handler: handlePalaceRoute},
		{Name: SyscallPalaceRecordRouteResult, Handler: handlePalaceRecordRouteResult},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	register(SyscallDefinition{Name: SyscallPalaceListRooms, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceListRooms})
	register(SyscallDefinition{Name: SyscallPalaceListAnchors, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceListAnchors})
	register(SyscallDefinition{Name: SyscallPalaceListRoutes, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceListRoutes})
	register(SyscallDefinition{Name: SyscallPalaceGetRoom, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceGetRoom})
	register(SyscallDefinition{Name: SyscallPalaceGetAnchor, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceGetAnchor})
	register(SyscallDefinition{Name: SyscallPalaceGetRoute, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handlePalaceGetRoute})
}

func handlePalaceCreateRoom(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	now := kernel.clock.Now()
	room, err := palace.NewMemoryRoom(palace.MemoryRoomInput{
		RoomID:      kernel.ids.NextID("room"),
		WorkspaceID: request.WorkspaceID,
		Name:        stringInput(request.Input, "name"),
		Description: stringInput(request.Input, "description"),
		DomainTags:  stringSliceInputDefault(request.Input, "domain_tags"),
		CreatedAt:   now,
		Metadata:    mapInput(request.Input, "metadata"),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryRoomCreated, room.RoomID, "", capabilityRefs, room)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	room.JournalRefs = []string{event.EventID}
	kernel.palace.CreateRoom(room)
	kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: room.RoomID, JournalEvent: event.EventID, Output: room}
}

func handlePalaceUpdateRoom(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	roomID := stringInput(request.Input, "room_id")
	room, ok := kernel.palace.GetRoom(roomID)
	if !ok || room.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if description := stringInput(request.Input, "description"); description != "" {
		room.Description = description
	}
	if tags, ok := stringSliceInput(request.Input, "domain_tags"); ok {
		room.DomainTags = tags
	}
	room.UpdatedAt = kernel.clock.Now()
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryRoomUpdated, room.RoomID, "", capabilityRefs, room)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	room.JournalRefs = append(room.JournalRefs, event.EventID)
	kernel.palace.UpdateRoom(room)
	kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: room.RoomID, JournalEvent: event.EventID, Output: room}
}

func handlePalaceLinkRooms(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	roomID := stringInput(request.Input, "room_id")
	linkedRoomID := stringInput(request.Input, "linked_room_id")
	if roomID == "" || linkedRoomID == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	room, ok := kernel.palace.GetRoom(roomID)
	if !ok || room.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	linked, ok := kernel.palace.GetRoom(linkedRoomID)
	if !ok || linked.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryRoomsLinked, roomID, "", capabilityRefs, map[string]string{"room_id": roomID, "linked_room_id": linkedRoomID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	room, linked, err = kernel.palace.LinkRooms(roomID, linkedRoomID, kernel.clock.Now(), event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	kernel.objects.putObject(memoryRoomObject(linked, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: roomID, JournalEvent: event.EventID, Output: room}
}

func handlePalaceCreateAnchor(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	now := kernel.clock.Now()
	anchor, err := palace.NewMemoryAnchor(palace.MemoryAnchorInput{
		AnchorID:     kernel.ids.NextID("anchor"),
		WorkspaceID:  request.WorkspaceID,
		RoomID:       stringInput(request.Input, "room_id"),
		Label:        stringInput(request.Input, "label"),
		ObjectRefs:   stringSliceInputDefault(request.Input, "object_refs"),
		Keywords:     stringSliceInputDefault(request.Input, "keywords"),
		Tags:         stringSliceInputDefault(request.Input, "tags"),
		SourceRefs:   stringSliceInputDefault(request.Input, "source_refs"),
		EmbeddingRef: stringInput(request.Input, "embedding_ref"),
		CreatedAt:    now,
		Metadata:     mapInput(request.Input, "metadata"),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryAnchorCreated, anchor.AnchorID, "", capabilityRefs, anchor)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	anchor.JournalRefs = []string{event.EventID}
	if err := kernel.palace.CreateAnchor(anchor); err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	kernel.objects.putObject(memoryAnchorObject(anchor, request.ActorID, capabilityRefs))
	if room, ok := kernel.palace.GetRoom(anchor.RoomID); ok {
		room.JournalRefs = appendUnique(room.JournalRefs, event.EventID)
		kernel.palace.UpdateRoom(room)
		kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: anchor.AnchorID, JournalEvent: event.EventID, Output: anchor}
}

func handlePalaceUpdateAnchor(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	anchorID := stringInput(request.Input, "anchor_id")
	anchor, ok := kernel.palace.GetAnchor(anchorID)
	if !ok || anchor.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if label := stringInput(request.Input, "label"); label != "" {
		anchor.Label = label
	}
	if keywords, ok := stringSliceInput(request.Input, "keywords"); ok {
		anchor.Keywords = keywords
	}
	if tags, ok := stringSliceInput(request.Input, "tags"); ok {
		anchor.Tags = tags
	}
	anchor.UpdatedAt = kernel.clock.Now()
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryAnchorUpdated, anchor.AnchorID, "", capabilityRefs, anchor)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	anchor.JournalRefs = append(anchor.JournalRefs, event.EventID)
	if err := kernel.palace.UpdateAnchor(anchor); err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	kernel.objects.putObject(memoryAnchorObject(anchor, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: anchor.AnchorID, JournalEvent: event.EventID, Output: anchor}
}

func handlePalaceLinkAnchor(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	anchorID := stringInput(request.Input, "anchor_id")
	objectRefs := stringSliceInputDefault(request.Input, "object_refs")
	sourceRefs := stringSliceInputDefault(request.Input, "source_refs")
	if anchorID == "" || len(objectRefs)+len(sourceRefs) == 0 {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	existing, ok := kernel.palace.GetAnchor(anchorID)
	if !ok || existing.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventMemoryAnchorLinked, anchorID, "", capabilityRefs, map[string]any{"anchor_id": anchorID, "object_refs": objectRefs, "source_refs": sourceRefs})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	anchor, err := kernel.palace.LinkAnchor(anchorID, objectRefs, sourceRefs, kernel.clock.Now(), event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	kernel.objects.putObject(memoryAnchorObject(anchor, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: anchor.AnchorID, JournalEvent: event.EventID, Output: anchor}
}

func handlePalaceRoute(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	caseID := stringInput(request.Input, "case_id")
	var cp CasePacket
	if caseID != "" {
		var err error
		cp, err = kernel.requireOpenCase(request)
		if err != nil {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
		}
	}
	query := palace.RouteQuery{
		CaseID:               caseID,
		WorkspaceID:          request.WorkspaceID,
		QueryText:            stringInput(request.Input, "query_text"),
		RouteReason:          stringInput(request.Input, "route_reason"),
		StartRoomID:          stringInput(request.Input, "start_room_id"),
		Tags:                 stringSliceInputDefault(request.Input, "tags"),
		ObjectTypePreference: stringInput(request.Input, "object_type_preference"),
	}
	route, updatedRooms, err := kernel.palace.CreateRoute(query, kernel.ids.NextID("route"), request.ActorID, kernel.clock.Now(), func() string {
		return kernel.ids.NextID("candidate")
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventPalaceRouteCreated, route.RouteID, caseID, capabilityRefs, route)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	route.JournalRefs = []string{event.EventID}
	kernel.palace.UpdateRoute(route)
	kernel.objects.putObject(palaceRouteObject(route, capabilityRefs))
	for _, candidate := range route.CandidateObjects {
		kernel.objects.putObject(candidateObject(candidate, capabilityRefs))
	}
	for _, room := range updatedRooms {
		room.JournalRefs = appendUnique(room.JournalRefs, event.EventID)
		kernel.palace.UpdateRoom(room)
		kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	}
	if caseID != "" {
		cp.PalaceRouteRefs = appendUnique(cp.PalaceRouteRefs, route.RouteID)
		for _, candidate := range route.CandidateObjects {
			cp.CandidateObjectRefs = appendUnique(cp.CandidateObjectRefs, candidate.CandidateID)
		}
		cp.RetrievalSummary = route.QueryText
		cp.JournalRefs = append(cp.JournalRefs, event.EventID)
		kernel.putUpdatedCase(cp, event.EventID)
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: route.RouteID, JournalEvent: event.EventID, Output: route}
}

func handlePalaceRecordRouteResult(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	routeID := stringInput(request.Input, "route_id")
	candidateID := stringInput(request.Input, "candidate_id")
	status := stringInput(request.Input, "result_status")
	if routeID == "" || candidateID == "" || status == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	existing, ok := kernel.palace.GetRoute(routeID)
	if !ok || existing.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	event, err := kernel.appendPalaceEvent(request, JournalEventPalaceRouteResultRecorded, routeID, "", capabilityRefs, request.Input)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	route, updatedRooms, err := kernel.palace.RecordRouteResult(routeID, candidateID, status, kernel.clock.Now(), event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapPalaceError(err)}
	}
	kernel.objects.putObject(palaceRouteObject(route, capabilityRefs))
	for _, room := range updatedRooms {
		kernel.objects.putObject(memoryRoomObject(room, request.ActorID, capabilityRefs))
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: routeID, JournalEvent: event.EventID, Output: route}
}

func handlePalaceListRooms(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.palace.ListRooms(request.WorkspaceID)}
}

func handlePalaceListAnchors(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.palace.ListAnchors(request.WorkspaceID)}
}

func handlePalaceListRoutes(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.palace.ListRoutes(request.WorkspaceID)}
}

func handlePalaceGetRoom(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	room, ok := kernel.palace.GetRoom(stringInput(request.Input, "room_id"))
	if !ok || room.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: room.RoomID, Output: room}
}

func handlePalaceGetAnchor(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	anchor, ok := kernel.palace.GetAnchor(stringInput(request.Input, "anchor_id"))
	if !ok || anchor.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: anchor.AnchorID, Output: anchor}
}

func handlePalaceGetRoute(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	route, ok := kernel.palace.GetRoute(stringInput(request.Input, "route_id"))
	if !ok || route.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: route.RouteID, Output: route}
}

func (k *Kernel) appendPalaceEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any) (JournalEvent, error) {
	return k.journal.Append(JournalEvent{
		EventType:      eventType,
		Timestamp:      k.clock.Now(),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(output),
		ObjectRefs:     []string{objectID},
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
}

func memoryRoomObject(room palace.MemoryRoom, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       room.RoomID,
		ObjectType:     ObjectTypeMemoryRoom,
		WorkspaceID:    room.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityCommitted,
		State: map[string]any{
			"name":             room.Name,
			"description":      room.Description,
			"domain_tags":      append([]string(nil), room.DomainTags...),
			"anchor_refs":      append([]string(nil), room.AnchorRefs...),
			"linked_room_refs": append([]string(nil), room.LinkedRoomRefs...),
			"route_stats":      routeStatsState(room.RouteStats),
		},
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       room.CreatedAt,
		UpdatedAt:       room.UpdatedAt,
		JournalRefs:     append([]string(nil), room.JournalRefs...),
	}
}

func memoryAnchorObject(anchor palace.MemoryAnchor, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       anchor.AnchorID,
		ObjectType:     ObjectTypeMemoryAnchor,
		WorkspaceID:    anchor.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityCommitted,
		State: map[string]any{
			"room_id":     anchor.RoomID,
			"label":       anchor.Label,
			"object_refs": append([]string(nil), anchor.ObjectRefs...),
			"keywords":    append([]string(nil), anchor.Keywords...),
			"tags":        append([]string(nil), anchor.Tags...),
		},
		SourceRefs:      append([]string(nil), anchor.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       anchor.CreatedAt,
		UpdatedAt:       anchor.UpdatedAt,
		JournalRefs:     append([]string(nil), anchor.JournalRefs...),
	}
}

func palaceRouteObject(route palace.PalaceRoute, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       route.RouteID,
		ObjectType:     ObjectTypePalaceRoute,
		WorkspaceID:    route.WorkspaceID,
		OwnerID:        route.CreatedBy,
		AuthorityLevel: AuthorityCommitted,
		State: map[string]any{
			"case_id":               route.CaseID,
			"query_text":            route.QueryText,
			"start_room_id":         route.StartRoomID,
			"visited_room_ids":      append([]string(nil), route.VisitedRoomIDs...),
			"anchor_refs":           append([]string(nil), route.AnchorRefs...),
			"candidate_object_refs": candidateRefs(route.CandidateObjects),
			"route_score":           route.RouteScore,
			"route_strategy":        route.RouteStrategy,
		},
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       route.CreatedAt,
		UpdatedAt:       route.CreatedAt,
		JournalRefs:     append([]string(nil), route.JournalRefs...),
	}
}

func candidateObject(candidate palace.CandidateObject, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       candidate.CandidateID,
		ObjectType:     ObjectTypeCandidate,
		WorkspaceID:    candidate.WorkspaceID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"source_object_id":  candidate.SourceObjectID,
			"source_type":       candidate.SourceType,
			"anchor_id":         candidate.AnchorID,
			"room_id":           candidate.RoomID,
			"relevance_score":   candidate.RelevanceScore,
			"retrieval_reason":  candidate.RetrievalReason,
			"candidate_summary": candidate.CandidateSummary,
		},
		SourceRefs:      append([]string(nil), candidate.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       candidate.CreatedAt,
		UpdatedAt:       candidate.CreatedAt,
	}
}

func routeStatsState(stats palace.RoomRouteStats) map[string]int {
	return map[string]int{
		"route_count":    stats.RouteCount,
		"success_count":  stats.SuccessCount,
		"rejected_count": stats.RejectedCount,
	}
}

func candidateRefs(candidates []palace.CandidateObject) []string {
	refs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		refs = append(refs, candidate.CandidateID)
	}
	return refs
}

func mapPalaceError(err error) error {
	switch {
	case errors.Is(err, palace.ErrRoomNotFound), errors.Is(err, palace.ErrAnchorNotFound), errors.Is(err, palace.ErrRouteNotFound):
		return ErrObjectNotFound
	case errors.Is(err, palace.ErrInvalidRoom), errors.Is(err, palace.ErrInvalidAnchor), errors.Is(err, palace.ErrInvalidRoute), errors.Is(err, palace.ErrInvalidCandidate):
		return ErrInvalidInput
	default:
		return err
	}
}
