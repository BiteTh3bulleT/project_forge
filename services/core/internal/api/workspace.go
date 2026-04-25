package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

// --- Chat ---

func (s *Server) handleChatThreadsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.chat.ListThreads(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	_ = json.NewDecoder(r.Body).Decode(&body)
	t, err := s.chat.CreateThread(ctx, body.Title, body.DossierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	t, err := s.chat.UpdateThreadTitle(ctx, id, body.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.InitiatingSource) == "" {
		body.InitiatingSource = "chat"
	}
	j, err := s.jobs.Create(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	meta := map[string]any{"jobId": j.ID, "templateId": body.TemplateID}
	_, err = s.chat.AppendMessage(ctx, threadID, "system", "Job queued: "+j.ID+" ("+j.Title+")", meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20) // 25MB hard request cap.
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		http.Error(w, "invalid multipart body", http.StatusBadRequest)
		return
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
		MaxBytes:   20 << 20,
		DefaultExt: filepath.Ext(filename),
		Metadata: map[string]any{
			"threadId": threadID,
			"source":   "chat-upload",
			"filename": filename,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	_ = json.NewDecoder(r.Body).Decode(&body)
	b, err := s.canvas.CreateBoard(ctx, body.Title, body.DossierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	n, err := s.canvas.PatchNote(ctx, boardID, noteID, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Workbench (artifacts) ---

func (s *Server) handleArtifactsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	job := strings.TrimSpace(r.URL.Query().Get("jobId"))
	var jobPtr *string
	if job != "" {
		jobPtr = &job
	}
	list, err := s.artifacts.List(ctx, jobPtr, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	body, art, textual, err := s.artifacts.ReadArtifactText(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":       art,
		"textual":        textual,
		"content":        body,
		"previewLimited": !textual,
	})
}

func workspaceIDFromPath(path string) string {
	base := strings.TrimSpace(filepath.Base(strings.TrimSpace(path)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "workspace:default"
	}
	return "workspace:" + base
}
