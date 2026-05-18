package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"forge/projectforge/services/core/internal/aios/domain"
)

// SemanticLinkRepository
func (s *SQLiteSemanticStore) CreateLinkWithKinds(ctx context.Context, link domain.SemanticLink, sourceKind, targetKind string) error {
	prevMeta := s.meta
	defer func() { s.meta = prevMeta }()
	return s.CreateLink(link)
}

func (s *SQLiteSemanticStore) GetByIDLink(ctx context.Context, id string) (domain.SemanticLink, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, type, source_id, target_id, workspace_id, lane_id, selected_paths_json, confidence, provenance_json, created_at
FROM semantic_links WHERE id = ?`, id)
	var link domain.SemanticLink
	var typ, provRaw, selected string
	if err := row.Scan(&link.ID, &typ, &link.SourceID, &link.TargetID, &link.Scope.WorkspaceID, &link.Scope.LaneID, &selected, &link.Confidence, &provRaw, &link.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SemanticLink{}, false, nil
		}
		return domain.SemanticLink{}, false, err
	}
	link.Type = domain.SemanticLinkType(typ)
	link.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(provRaw), &link.Provenance)
	return link, true, nil
}

func (s *SQLiteSemanticStore) ListBySource(ctx context.Context, sourceID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `source_id = ?`, []any{sourceID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByTarget(ctx context.Context, targetID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `target_id = ?`, []any{targetID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListNeighborhood(ctx context.Context, objectID string, scope ScopeFilter, depth, limit int) ([]domain.SemanticLink, error) {
	if depth <= 1 {
		return s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{objectID, objectID}, scope, limit)
	}
	oneHop, err := s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{objectID, objectID}, scope, limit)
	if err != nil {
		return nil, err
	}
	seen := map[string]domain.SemanticLink{}
	nextObjects := map[string]struct{}{}
	for _, l := range oneHop {
		seen[l.ID] = l
		nextObjects[l.SourceID] = struct{}{}
		nextObjects[l.TargetID] = struct{}{}
	}
	for oid := range nextObjects {
		links, err := s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{oid, oid}, scope, limit)
		if err != nil {
			return nil, err
		}
		for _, l := range links {
			seen[l.ID] = l
		}
	}
	out := make([]domain.SemanticLink, 0, len(seen))
	for _, l := range seen {
		out = append(out, l)
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (s *SQLiteSemanticStore) ListByTypeLinks(ctx context.Context, typ domain.SemanticLinkType, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `type = ?`, []any{string(typ)}, scope, limit)
}
