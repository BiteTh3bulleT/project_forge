package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/canvas"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/jobs"
)

const (
	workspaceJSONRequestBodyLimit       = 1 << 20
	chatAttachmentUploadRequestLimit    = 25 << 20
	chatAttachmentUploadArtifactMaxSize = 20 << 20
)

var errWorkspaceRequestBodyTooLarge = errors.New("workspace json request body too large")

// --- Chat ---

func (s *Server) handleChatThreadsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.chat.ListThreads(ctx, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"threads": list})
}

func (s *Server) handleChatThreadCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Title     string `json:"title"`
		DossierID *int64 `json:"dossierId"`
	}
	if err := decodeOptionalWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	t, err := s.chat.CreateThread(ctx, body.Title, body.DossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(ctx, "chat.thread.created", map[string]any{"threadId": t.ID})
	writeJSON(w, http.StatusOK, map[string]any{"thread": t})
}

func (s *Server) handleChatThreadGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	d, err := s.chat.GetThread(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, chatThreadDetailForAPI(d))
}

func chatThreadDetailForAPI(d *chat.ThreadDetail) *chat.ThreadDetail {
	if d == nil {
		return nil
	}
	out := *d
	if len(d.Messages) == 0 {
		out.Messages = []chat.Message{}
		return &out
	}
	out.Messages = make([]chat.Message, len(d.Messages))
	for i, msg := range d.Messages {
		next := msg
		next.Metadata = chatMetadataForAPI(msg.Metadata)
		out.Messages[i] = next
	}
	return &out
}

func chatMetadataForAPI(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(meta))
	for key, value := range meta {
		switch key {
		case "toolManifest":
			out["toolManifestSummary"] = omittedJSONSummary(value)
		case "toolGatewayActivity":
			out[key] = chatToolGatewayActivityForAPI(value)
		default:
			out[key] = value
		}
	}
	return out
}

func chatToolGatewayActivityForAPI(value any) any {
	activity, ok := value.(map[string]any)
	if !ok || activity == nil {
		return value
	}
	out := make(map[string]any, len(activity))
	for key, nested := range activity {
		if key == "toolManifest" {
			out["toolManifestSummary"] = omittedJSONSummary(nested)
			continue
		}
		out[key] = nested
	}
	return out
}

func omittedJSONSummary(value any) map[string]any {
	summary := map[string]any{
		"omitted": true,
		"reason":  "omitted_from_chat_thread_api_response",
	}
	switch typed := value.(type) {
	case []any:
		summary["count"] = len(typed)
	case map[string]any:
		summary["keys"] = len(typed)
	}
	return summary
}

func (s *Server) handleChatThreadPatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	t, err := s.chat.UpdateThreadTitle(ctx, id, body.Title)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(ctx, "chat.thread.renamed", map[string]any{"threadId": id})
	writeJSON(w, http.StatusOK, map[string]any{"thread": t})
}

func (s *Server) handleChatThreadDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.chat.DeleteThread(ctx, id); err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.log.Emit(ctx, "chat.thread.deleted", map[string]any{"threadId": id})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChatJobCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	threadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad thread id", http.StatusBadRequest)
		return
	}
	var body jobs.CreateRequest
	if err := decodeWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	if strings.TrimSpace(body.InitiatingSource) == "" {
		body.InitiatingSource = "chat"
	}
	j, err := s.jobs.Create(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	meta := map[string]any{"jobId": j.ID, "templateId": body.TemplateID}
	_, err = s.chat.AppendMessage(ctx, threadID, "system", "Job queued: "+j.ID+" ("+j.Title+")", meta)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.log.Emit(ctx, "chat.job.queued", map[string]any{"threadId": threadID, "jobId": j.ID})
	writeJSON(w, http.StatusOK, map[string]any{"job": j})
}

func (s *Server) handleChatAttachmentUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	threadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad thread id", http.StatusBadRequest)
		return
	}
	if _, err := s.chat.GetThread(ctx, threadID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, chatAttachmentUploadRequestLimit)
	if err := r.ParseMultipartForm(chatAttachmentUploadRequestLimit); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			http.Error(w, "chat attachment request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid multipart body", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer f.Close()

	filename := strings.TrimSpace(fh.Filename)
	if filename == "" {
		filename = "upload.bin"
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = filename
	}
	mime := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if mime == "" {
		mime = "application/octet-stream"
	}
	now := time.Now().UTC().Format("20060102")
	art, bytesWritten, err := s.artifacts.CreateFileArtifact(ctx, artifacts.CreateFileArtifactRequest{
		Type:       "chat_attachment",
		Title:      title,
		FileName:   filename,
		Subdir:     filepath.Join("chat", fmt.Sprintf("thread-%d", threadID), now),
		Reader:     f,
		MimeType:   mime,
		MaxBytes:   chatAttachmentUploadArtifactMaxSize,
		DefaultExt: filepath.Ext(filename),
		Metadata: map[string]any{
			"threadId": threadID,
			"source":   "chat-upload",
			"filename": filename,
		},
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	previewText := ""
	if strings.HasPrefix(strings.ToLower(art.MimeType), "text/") || strings.HasSuffix(strings.ToLower(art.FilePath), ".md") || strings.HasSuffix(strings.ToLower(art.FilePath), ".json") || strings.HasSuffix(strings.ToLower(art.FilePath), ".txt") {
		if content, _, textual, readErr := s.artifacts.ReadArtifactText(ctx, art.ID); readErr == nil && textual {
			runes := []rune(content)
			if len(runes) > 2000 {
				previewText = string(runes[:2000]) + "\n…"
			} else {
				previewText = content
			}
		}
	}
	_ = s.log.Emit(ctx, "chat.attachment.uploaded", map[string]any{"threadId": threadID, "artifactId": art.ID, "bytes": bytesWritten})
	if s.auditSvc != nil {
		meta := requestAuditMetaForBackup(r, "", "", "", "artifact.upload")
		if strings.TrimSpace(meta.WorkspaceID) == "" {
			meta.WorkspaceID = workspaceIDFromPath(s.cfg.WorkspaceDir)
		}
		_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
			CorrelationID: meta.CorrelationID,
			Category:      "artifact",
			Action:        "artifact.uploaded",
			Actor:         "api",
			SubjectType:   "artifact",
			SubjectID:     strconv.FormatInt(art.ID, 10),
			Outcome:       "ok",
			Summary:       "chat attachment stored as artifact",
			Payload: requestAuditPayload(map[string]any{
				"threadId":    threadID,
				"artifactId":  art.ID,
				"bytes":       bytesWritten,
				"fileName":    filename,
				"mimeType":    mime,
				"requestPath": r.URL.Path,
			}, meta),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":    art,
		"bytes":       bytesWritten,
		"previewText": previewText,
	})
}

// --- Canvas ---

func (s *Server) handleCanvasBoardsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.canvas.ListBoards(ctx, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"boards": list})
}

func (s *Server) handleCanvasBoardCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Title     string `json:"title"`
		DossierID *int64 `json:"dossierId"`
	}
	if err := decodeOptionalWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	b, err := s.canvas.CreateBoard(ctx, body.Title, body.DossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"board": b})
}

func (s *Server) handleCanvasBoardGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	b, err := s.canvas.GetBoard(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleCanvasBoardDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.canvas.DeleteBoard(ctx, id); err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCanvasNoteCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	boardID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad board id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title  string  `json:"title"`
		Body   string  `json:"body"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}
	if err := decodeWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	if body.Width <= 0 {
		body.Width = 260
	}
	if body.Height <= 0 {
		body.Height = 180
	}
	n, err := s.canvas.CreateNote(ctx, boardID, body.Title, body.Body, body.X, body.Y, body.Width, body.Height)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": n})
}

func (s *Server) handleCanvasNotePatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	boardID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad board id", http.StatusBadRequest)
		return
	}
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteId"), 10, 64)
	if err != nil {
		http.Error(w, "bad note id", http.StatusBadRequest)
		return
	}
	var body canvas.PatchNote
	if err := decodeWorkspaceJSONBody(r, &body); err != nil {
		writeWorkspaceDecodeError(w, err)
		return
	}
	n, err := s.canvas.PatchNote(ctx, boardID, noteID, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": n})
}

func (s *Server) handleCanvasNoteDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	boardID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad board id", http.StatusBadRequest)
		return
	}
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteId"), 10, 64)
	if err != nil {
		http.Error(w, "bad note id", http.StatusBadRequest)
		return
	}
	if err := s.canvas.DeleteNote(ctx, boardID, noteID); err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Workbench (artifacts) ---

var errArtifactThreadScopeRequired = errors.New("artifact requires matching chat thread scope")

func (s *Server) handleArtifactsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	threadID, err := artifactThreadScopeFromRequest(r)
	if err != nil {
		http.Error(w, "bad threadId", http.StatusBadRequest)
		return
	}
	job := strings.TrimSpace(r.URL.Query().Get("jobId"))
	var jobPtr *string
	if job != "" {
		jobPtr = &job
	}
	list, err := s.artifacts.List(ctx, jobPtr, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	list, err = s.filterArtifactsForPublicList(ctx, list, threadID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": list})
}

func (s *Server) handleArtifactGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	a, err := s.artifacts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	threadID, err := artifactThreadScopeFromRequest(r)
	if err != nil {
		http.Error(w, "bad threadId", http.StatusBadRequest)
		return
	}
	if err := s.authorizeArtifactPublicRead(ctx, a, threadID); err != nil {
		if errors.Is(err, errArtifactThreadScopeRequired) {
			writeAPIRequestError(w, http.StatusForbidden, err)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleArtifactContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	art, err := s.artifacts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	threadID, err := artifactThreadScopeFromRequest(r)
	if err != nil {
		http.Error(w, "bad threadId", http.StatusBadRequest)
		return
	}
	if err := s.authorizeArtifactPublicRead(ctx, art, threadID); err != nil {
		if errors.Is(err, errArtifactThreadScopeRequired) {
			writeAPIRequestError(w, http.StatusForbidden, err)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	body, art, textual, err := s.artifacts.ReadArtifactText(ctx, id)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":       art,
		"textual":        textual,
		"content":        body,
		"previewLimited": !textual,
	})
}

func artifactThreadScopeFromRequest(r *http.Request) (*int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("threadId"))
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid threadId")
	}
	return &id, nil
}

func (s *Server) filterArtifactsForPublicList(ctx context.Context, list []artifacts.Artifact, threadID *int64) ([]artifacts.Artifact, error) {
	out := make([]artifacts.Artifact, 0, len(list))
	for _, art := range list {
		if art.Type != "chat_attachment" {
			out = append(out, art)
			continue
		}
		if err := s.authorizeArtifactPublicRead(ctx, &art, threadID); err != nil {
			if errors.Is(err, errArtifactThreadScopeRequired) {
				continue
			}
			return nil, err
		}
		out = append(out, art)
	}
	return out, nil
}

func (s *Server) authorizeArtifactPublicRead(ctx context.Context, art *artifacts.Artifact, threadID *int64) error {
	if art == nil || art.Type != "chat_attachment" {
		return nil
	}
	if threadID == nil {
		return errArtifactThreadScopeRequired
	}
	artifactThreadID, ok := artifactChatThreadID(art)
	if !ok || artifactThreadID != *threadID {
		return errArtifactThreadScopeRequired
	}
	linked, err := s.threadHasAttachment(ctx, *threadID, art.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errArtifactThreadScopeRequired
		}
		return err
	}
	if !linked {
		return errArtifactThreadScopeRequired
	}
	return nil
}

func artifactChatThreadID(art *artifacts.Artifact) (int64, bool) {
	if art == nil || art.Type != "chat_attachment" || len(art.Metadata) == 0 {
		return 0, false
	}
	var meta map[string]any
	if err := json.Unmarshal(art.Metadata, &meta); err != nil {
		return 0, false
	}
	return metadataInt64(meta["threadId"])
}

func metadataInt64(raw any) (int64, bool) {
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return int64(v), true
		}
	case int64:
		if v > 0 {
			return v, true
		}
	case int:
		if v > 0 {
			return int64(v), true
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func (s *Server) threadHasAttachment(ctx context.Context, threadID, artifactID int64) (bool, error) {
	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		return false, err
	}
	for _, msg := range th.Messages {
		for _, attachedID := range messageAttachmentIDs(msg.Metadata) {
			if attachedID == artifactID {
				return true, nil
			}
		}
	}
	return false, nil
}

func workspaceIDFromPath(path string) string {
	base := strings.TrimSpace(filepath.Base(strings.TrimSpace(path)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "workspace:default"
	}
	return "workspace:" + base
}

func decodeWorkspaceJSONBody(r *http.Request, target any) error {
	raw, err := readWorkspaceRequestBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalWorkspaceJSONBody(r *http.Request, target any) error {
	raw, err := readWorkspaceRequestBody(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func readWorkspaceRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, workspaceJSONRequestBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > workspaceJSONRequestBodyLimit {
		return nil, errWorkspaceRequestBodyTooLarge
	}
	return raw, nil
}

func writeWorkspaceDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errWorkspaceRequestBodyTooLarge) {
		http.Error(w, "workspace json request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}
