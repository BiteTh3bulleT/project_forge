package canvas

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBoardLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := New(testDB(t))

	dossierID := int64(42)
	board, err := svc.CreateBoard(ctx, "  Incident Board  ", &dossierID)
	if err != nil {
		t.Fatalf("CreateBoard: %v", err)
	}
	if board.Title != "Incident Board" {
		t.Fatalf("CreateBoard title = %q, want trimmed title", board.Title)
	}
	if board.DossierID == nil || *board.DossierID != dossierID {
		t.Fatalf("CreateBoard dossier id = %v, want %d", board.DossierID, dossierID)
	}

	defaultBoard, err := svc.CreateBoard(ctx, "   ", nil)
	if err != nil {
		t.Fatalf("CreateBoard with blank title: %v", err)
	}
	if defaultBoard.Title != "Board" {
		t.Fatalf("blank CreateBoard title = %q, want Board", defaultBoard.Title)
	}

	boards, err := svc.ListBoards(ctx, 0)
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if len(boards) != 2 {
		t.Fatalf("ListBoards returned %d boards, want 2", len(boards))
	}

	note, err := svc.CreateNote(ctx, board.ID, "  First note  ", "body", 10, 20, 300, 180)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if note.Title != "  First note  " {
		t.Fatalf("CreateNote returned title = %q, want original title", note.Title)
	}
	if note.Pinned {
		t.Fatalf("CreateNote returned pinned = true, want false")
	}
	if len(note.Links) != 0 {
		t.Fatalf("CreateNote links = %v, want empty links", note.Links)
	}

	detail, err := svc.GetBoard(ctx, board.ID)
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if detail.Title != "Incident Board" {
		t.Fatalf("GetBoard title = %q, want Incident Board", detail.Title)
	}
	if len(detail.Notes) != 1 {
		t.Fatalf("GetBoard notes length = %d, want 1", len(detail.Notes))
	}
	got := detail.Notes[0]
	if got.Title != "First note" {
		t.Fatalf("persisted note title = %q, want trimmed title", got.Title)
	}
	if got.Body != note.Body || got.Color != note.Color {
		t.Fatalf("persisted note body/color = %q/%q, want %q/%q", got.Body, got.Color, note.Body, note.Color)
	}
}

func TestPatchNoteUpdatesOnlyProvidedFields(t *testing.T) {
	ctx := context.Background()
	svc := New(testDB(t))

	board, err := svc.CreateBoard(ctx, "Board", nil)
	if err != nil {
		t.Fatalf("CreateBoard: %v", err)
	}
	note, err := svc.CreateNote(ctx, board.ID, "Original", "body", 1, 2, 300, 200)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	title := "  Updated  "
	pinned := true
	links := []map[string]any{{"kind": "job", "id": "job-1"}}
	patched, err := svc.PatchNote(ctx, board.ID, note.ID, PatchNote{
		Title:  &title,
		Pinned: &pinned,
		Links:  links,
	})
	if err != nil {
		t.Fatalf("PatchNote: %v", err)
	}
	if patched.Title != "Updated" {
		t.Fatalf("PatchNote title = %q, want trimmed title", patched.Title)
	}
	if patched.Body != "body" || patched.X != 1 || patched.Y != 2 || patched.Width != 300 || patched.Height != 200 {
		t.Fatalf("PatchNote changed unspecified fields: %+v", patched)
	}
	if !patched.Pinned {
		t.Fatalf("PatchNote pinned = false, want true")
	}
	if len(patched.Links) != 1 || patched.Links[0]["kind"] != "job" || patched.Links[0]["id"] != "job-1" {
		t.Fatalf("PatchNote links = %#v, want job link", patched.Links)
	}

	detail, err := svc.GetBoard(ctx, board.ID)
	if err != nil {
		t.Fatalf("GetBoard after patch: %v", err)
	}
	if len(detail.Notes) != 1 {
		t.Fatalf("GetBoard notes length = %d, want 1", len(detail.Notes))
	}
	persisted := detail.Notes[0]
	if persisted.Title != "Updated" || !persisted.Pinned {
		t.Fatalf("persisted patched note = %+v, want updated pinned note", persisted)
	}
	if len(persisted.Links) != 1 || persisted.Links[0]["id"] != "job-1" {
		raw, _ := json.Marshal(persisted.Links)
		t.Fatalf("persisted links = %s, want job-1 link", raw)
	}
}

func TestDeleteNote(t *testing.T) {
	tests := []struct {
		name    string
		missing bool
		wantErr bool
	}{
		{name: "existing note is removed"},
		{name: "missing note returns error", missing: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svc := New(testDB(t))
			board, err := svc.CreateBoard(ctx, "Board", nil)
			if err != nil {
				t.Fatalf("CreateBoard: %v", err)
			}
			note, err := svc.CreateNote(ctx, board.ID, "Note", "body", 0, 0, 10, 10)
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}

			noteID := note.ID
			if tt.missing {
				noteID++
			}
			err = svc.DeleteNote(ctx, board.ID, noteID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("DeleteNote error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteNote: %v", err)
			}
			detail, err := svc.GetBoard(ctx, board.ID)
			if err != nil {
				t.Fatalf("GetBoard after DeleteNote: %v", err)
			}
			if len(detail.Notes) != 0 {
				t.Fatalf("notes after DeleteNote = %d, want 0", len(detail.Notes))
			}
		})
	}
}

func TestDeleteBoardCascadesNotes(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	svc := New(db)
	board, err := svc.CreateBoard(ctx, "Board", nil)
	if err != nil {
		t.Fatalf("CreateBoard: %v", err)
	}
	if _, err := svc.CreateNote(ctx, board.ID, "Note", "body", 0, 0, 10, 10); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	if err := svc.DeleteBoard(ctx, board.ID); err != nil {
		t.Fatalf("DeleteBoard: %v", err)
	}
	if _, err := svc.GetBoard(ctx, board.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetBoard deleted board error = %v, want sql.ErrNoRows", err)
	}
	var notes int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM canvas_notes`).Scan(&notes); err != nil {
		t.Fatalf("count notes: %v", err)
	}
	if notes != 0 {
		t.Fatalf("notes after DeleteBoard = %d, want 0", notes)
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mustExec(t, db, `PRAGMA foreign_keys = ON`)
	mustExec(t, db, `
CREATE TABLE canvas_boards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  dossier_id INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)`)
	mustExec(t, db, `
CREATE TABLE canvas_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  board_id INTEGER NOT NULL REFERENCES canvas_boards(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  x REAL NOT NULL DEFAULT 0,
  y REAL NOT NULL DEFAULT 0,
  width REAL NOT NULL DEFAULT 260,
  height REAL NOT NULL DEFAULT 180,
  pinned INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL DEFAULT '',
  links_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)`)
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
