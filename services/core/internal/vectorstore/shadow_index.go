package vectorstore

import (
	"context"
	"strings"
	"time"
)

type ShadowIndexConfig struct {
	Enabled          bool
	Collection       string
	VectorSize       int
	MaxPayloadBytes  int
	EnsureCollection bool
}

type ShadowIndexService struct {
	store VectorStore
	cfg   ShadowIndexConfig
}

type ShadowIndexRequest struct {
	Vector  []float64
	Payload SafePayload
	PointID string
}

type ShadowIndexResult struct {
	Indexed    bool
	PointID    string
	Collection string
	Skipped    bool
	Reason     string
}

func NewShadowIndexService(store VectorStore, cfg ShadowIndexConfig) *ShadowIndexService {
	if strings.TrimSpace(cfg.Collection) == "" {
		cfg.Collection = DefaultCollection
	}
	if cfg.MaxPayloadBytes <= 0 {
		cfg.MaxPayloadBytes = 8192
	}
	return &ShadowIndexService{store: store, cfg: cfg}
}

func (s *ShadowIndexService) Index(ctx context.Context, req ShadowIndexRequest) (ShadowIndexResult, error) {
	collection := strings.TrimSpace(s.cfg.Collection)
	if collection == "" {
		collection = DefaultCollection
	}
	if !s.cfg.Enabled {
		return ShadowIndexResult{Collection: collection, Skipped: true, Reason: "disabled"}, nil
	}
	if s.store == nil {
		return ShadowIndexResult{}, ErrInvalidConfig
	}
	payload := req.Payload.Normalized()
	if payload.CreatedAtUnix == 0 {
		payload.CreatedAtUnix = time.Now().UTC().Unix()
	}
	if err := payload.Validate(s.cfg.MaxPayloadBytes); err != nil {
		return ShadowIndexResult{}, err
	}
	if err := ValidateVector(req.Vector, payload.EmbeddingDims); err != nil {
		return ShadowIndexResult{}, err
	}
	if s.cfg.VectorSize > 0 && len(req.Vector) != s.cfg.VectorSize {
		return ShadowIndexResult{}, ErrInvalidVectorDimensions
	}
	if s.cfg.EnsureCollection {
		if err := s.store.EnsureCollection(ctx, CollectionSpec{
			Name:       collection,
			VectorSize: len(req.Vector),
			Distance:   "Cosine",
		}); err != nil {
			return ShadowIndexResult{}, err
		}
	}
	pointID := strings.TrimSpace(req.PointID)
	if pointID == "" {
		pointID = DeterministicPointID(payload)
	}
	if err := s.store.UpsertVector(ctx, VectorPoint{
		Collection: collection,
		PointID:    pointID,
		Vector:     append([]float64(nil), req.Vector...),
		Payload:    payload,
	}); err != nil {
		return ShadowIndexResult{}, err
	}
	return ShadowIndexResult{Indexed: true, PointID: pointID, Collection: collection}, nil
}

func (s *ShadowIndexService) CanExecuteRetrieval() bool {
	return false
}

func (s *ShadowIndexService) CanCreateEmbeddings() bool {
	return false
}
