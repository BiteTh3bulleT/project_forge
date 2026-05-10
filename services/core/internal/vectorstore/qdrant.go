package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	qdrantResponseBodyLimit = 2 << 20
	qdrantErrorBodyLimit    = 4096
)

type QdrantClient struct {
	baseURL string
	client  *http.Client
}

func NewQdrantClient(baseURL string, timeout time.Duration) (*QdrantClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, ErrInvalidConfig
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, ErrInvalidConfig
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &QdrantClient{baseURL: baseURL, client: &http.Client{Timeout: timeout}}, nil
}

func (q *QdrantClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q.baseURL+"/readyz", nil)
	if err != nil {
		return err
	}
	res, err := q.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("qdrant health failed: %s", res.Status)
	}
	return nil
}

func (q *QdrantClient) EnsureCollection(ctx context.Context, spec CollectionSpec) error {
	name := strings.TrimSpace(spec.Name)
	if name == "" || spec.VectorSize <= 0 {
		return ErrInvalidCollection
	}
	distance := strings.TrimSpace(spec.Distance)
	if distance == "" {
		distance = "Cosine"
	}
	body := map[string]any{
		"vectors": map[string]any{
			"size":     spec.VectorSize,
			"distance": distance,
		},
	}
	return q.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(name), body, nil)
}

func (q *QdrantClient) UpsertVector(ctx context.Context, point VectorPoint) error {
	if strings.TrimSpace(point.Collection) == "" || strings.TrimSpace(point.PointID) == "" {
		return ErrInvalidPayload
	}
	if err := ValidateVector(point.Vector, point.Payload.EmbeddingDims); err != nil {
		return err
	}
	payload := point.Payload.Normalized()
	if err := payload.Validate(8192); err != nil {
		return err
	}
	body := map[string]any{
		"points": []map[string]any{{
			"id":      strings.TrimSpace(point.PointID),
			"vector":  point.Vector,
			"payload": payload,
		}},
	}
	path := "/collections/" + url.PathEscape(point.Collection) + "/points?wait=true"
	return q.doJSON(ctx, http.MethodPut, path, body, nil)
}

func (q *QdrantClient) SearchVector(ctx context.Context, req SearchRequest) (SearchResult, error) {
	if strings.TrimSpace(req.Collection) == "" {
		return SearchResult{}, ErrInvalidCollection
	}
	if err := ValidateVector(req.Vector, 0); err != nil {
		return SearchResult{}, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{
		"vector":       req.Vector,
		"limit":        limit,
		"with_payload": true,
	}
	var raw struct {
		Result []struct {
			ID      any         `json:"id"`
			Score   float64     `json:"score"`
			Payload SafePayload `json:"payload"`
		} `json:"result"`
	}
	path := "/collections/" + url.PathEscape(req.Collection) + "/points/search"
	if err := q.doJSON(ctx, http.MethodPost, path, body, &raw); err != nil {
		return SearchResult{}, err
	}
	out := SearchResult{Matches: make([]SearchMatch, 0, len(raw.Result))}
	for _, item := range raw.Result {
		out.Matches = append(out.Matches, SearchMatch{
			PointID: fmt.Sprint(item.ID),
			Score:   item.Score,
			Payload: item.Payload.Normalized(),
		})
	}
	return out, nil
}

func (q *QdrantClient) DeleteVector(ctx context.Context, collection, pointID string) error {
	collection = strings.TrimSpace(collection)
	pointID = strings.TrimSpace(pointID)
	if collection == "" || pointID == "" {
		return ErrInvalidPayload
	}
	body := map[string]any{
		"points": []string{pointID},
	}
	path := "/collections/" + url.PathEscape(collection) + "/points/delete?wait=true"
	return q.doJSON(ctx, http.MethodPost, path, body, nil)
}

func (q *QdrantClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := q.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := readQdrantErrorBody(res.Body)
		if err != nil {
			return fmt.Errorf("read qdrant error response: %w", err)
		}
		return fmt.Errorf("qdrant request failed: %s: %s", res.Status, body)
	}
	if out == nil {
		return nil
	}
	raw, err := readQdrantResponseBody(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func readQdrantResponseBody(body io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, qdrantResponseBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > qdrantResponseBodyLimit {
		return nil, fmt.Errorf("qdrant response too large: limit %d bytes", qdrantResponseBodyLimit)
	}
	return raw, nil
}

func readQdrantErrorBody(body io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(body, qdrantErrorBodyLimit+1))
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if len(raw) > qdrantErrorBodyLimit {
		text = strings.TrimSpace(string(raw[:qdrantErrorBodyLimit]))
		if text != "" {
			text += " "
		}
		text += "[truncated]"
	}
	return text, nil
}
