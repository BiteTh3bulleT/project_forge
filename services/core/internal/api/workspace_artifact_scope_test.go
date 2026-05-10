package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/artifacts"
)

func TestArtifactContentRequiresMatchingChatThreadScope(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx := context.Background()

	threadA, err := srv.chat.CreateThread(ctx, "thread a", nil)
	if err != nil {
		t.Fatalf("create thread a: %v", err)
	}
	threadB, err := srv.chat.CreateThread(ctx, "thread b", nil)
	if err != nil {
		t.Fatalf("create thread b: %v", err)
	}
	art := mustCreateLinkedChatAttachment(t, srv, threadA.ID, "secret.txt", "thread-a secret")

	for _, tc := range []struct {
		name string
		path string
	}{
		{"without thread scope", "/api/artifacts/" + strconv.FormatInt(art.ID, 10) + "/content"},
		{"with wrong thread scope", "/api/artifacts/" + strconv.FormatInt(art.ID, 10) + "/content?threadId=" + strconv.FormatInt(threadB.ID, 10)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := withRouteParam(httptest.NewRequest(http.MethodGet, tc.path, nil), "id", strconv.FormatInt(art.ID, 10))
			rr := httptest.NewRecorder()
			srv.handleArtifactContent(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected forbidden, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
			}
		})
	}

	getReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/artifacts/"+strconv.FormatInt(art.ID, 10), nil), "id", strconv.FormatInt(art.ID, 10))
	getRR := httptest.NewRecorder()
	srv.handleArtifactGet(getRR, getReq)
	if getRR.Code != http.StatusForbidden {
		t.Fatalf("expected unscoped artifact metadata read to be forbidden, got %d body=%s", getRR.Code, strings.TrimSpace(getRR.Body.String()))
	}

	req := withRouteParam(
		httptest.NewRequest(http.MethodGet, "/api/artifacts/"+strconv.FormatInt(art.ID, 10)+"/content?threadId="+strconv.FormatInt(threadA.ID, 10), nil),
		"id",
		strconv.FormatInt(art.ID, 10),
	)
	rr := httptest.NewRecorder()
	srv.handleArtifactContent(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected scoped content read success, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !strings.Contains(rr.Body.String(), "thread-a secret") {
		t.Fatalf("expected scoped content body, got %s", rr.Body.String())
	}
}

func TestArtifactsListHidesChatAttachmentsUnlessThreadScoped(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx := context.Background()

	threadA, err := srv.chat.CreateThread(ctx, "thread a", nil)
	if err != nil {
		t.Fatalf("create thread a: %v", err)
	}
	threadB, err := srv.chat.CreateThread(ctx, "thread b", nil)
	if err != nil {
		t.Fatalf("create thread b: %v", err)
	}
	chatArt := mustCreateLinkedChatAttachment(t, srv, threadA.ID, "secret.txt", "thread-a secret")
	publicArt, err := srv.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
		Type:     "report",
		Title:    "public.txt",
		FileName: "public.txt",
		Subdir:   "reports",
		Content:  "public artifact",
		MimeType: "text/plain",
	})
	if err != nil {
		t.Fatalf("create public artifact: %v", err)
	}

	global := artifactIDsFromList(t, srv, "/api/artifacts?limit=20")
	if containsArtifactID(global, chatArt.ID) {
		t.Fatalf("global artifact list exposed chat attachment %d: %#v", chatArt.ID, global)
	}
	if !containsArtifactID(global, publicArt.ID) {
		t.Fatalf("global artifact list omitted public artifact %d: %#v", publicArt.ID, global)
	}

	scoped := artifactIDsFromList(t, srv, "/api/artifacts?limit=20&threadId="+strconv.FormatInt(threadA.ID, 10))
	if !containsArtifactID(scoped, chatArt.ID) {
		t.Fatalf("matching thread list omitted chat attachment %d: %#v", chatArt.ID, scoped)
	}

	wrongThread := artifactIDsFromList(t, srv, "/api/artifacts?limit=20&threadId="+strconv.FormatInt(threadB.ID, 10))
	if containsArtifactID(wrongThread, chatArt.ID) {
		t.Fatalf("wrong thread list exposed chat attachment %d: %#v", chatArt.ID, wrongThread)
	}
}

func mustCreateLinkedChatAttachment(t *testing.T, srv *Server, threadID int64, name, content string) artifacts.Artifact {
	t.Helper()
	art, err := srv.artifacts.CreateTextArtifact(context.Background(), artifacts.CreateTextArtifactRequest{
		Type:     "chat_attachment",
		Title:    name,
		FileName: name,
		Subdir:   "chat",
		Content:  content,
		MimeType: "text/plain",
		Metadata: map[string]any{"threadId": threadID},
	})
	if err != nil {
		t.Fatalf("create chat attachment: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), threadID, "user", "attached", map[string]any{
		"attachments": []any{map[string]any{"artifactId": art.ID}},
	}); err != nil {
		t.Fatalf("append attachment message: %v", err)
	}
	return art
}

func artifactIDsFromList(t *testing.T, srv *Server, path string) []int64 {
	t.Helper()
	rr := httptest.NewRecorder()
	srv.handleArtifactsList(rr, httptest.NewRequest(http.MethodGet, path, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected artifact list success, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var resp struct {
		Artifacts []struct {
			ID int64 `json:"id"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode artifact list: %v body=%s", err, rr.Body.String())
	}
	out := make([]int64, 0, len(resp.Artifacts))
	for _, artifact := range resp.Artifacts {
		out = append(out, artifact.ID)
	}
	return out
}

func containsArtifactID(ids []int64, id int64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
