package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

func TestLoadSessionsAndGraph(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "chats", "chat_a.json"), `{"id":"chat_a","title":"A","kind":"teach","updatedAt":"2026-11-20T10:00:00Z","messages":[{"id":"m1","role":"user","content":"hi"}]}`)
	mustWrite(t, filepath.Join(dir, "chats", "broken.json"), `{not json`)
	mustWrite(t, filepath.Join(dir, "graph", "nodes.jsonl"), `{"id":"n1","name":"N","provenance":[]}`)
	mustWrite(t, filepath.Join(dir, "graph", "chains.jsonl"), "")
	mustWrite(t, filepath.Join(dir, "graph", "sources.json"), `{"src_1":{"id":"src_1","filePath":"notes.md","importedAt":"2026-11-01T00:00:00Z"}}`)

	sessions, skipped, err := LoadSessions(dir)
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "chat_a" {
		t.Fatalf("sessions = %+v", sessions)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}

	graph, err := LoadGraph(dir)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if graph.Name("n1") != "N" {
		t.Fatalf("graph node missing: %+v", graph)
	}
	if graph.Sources["src_1"].FilePath != "notes.md" {
		t.Fatalf("sources not loaded: %+v", graph.Sources)
	}
}

func TestLoadGraphMissingFilesIsEmpty(t *testing.T) {
	graph, err := LoadGraph(t.TempDir())
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if len(graph.Order) != 0 || len(graph.Chains) != 0 {
		t.Fatalf("expected empty graph: %+v", graph)
	}
}

func TestSaveAndLoadStudy(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 11, 21, 14, 5, 0, 0, time.UTC)
	rel, err := SaveStudy(dir, domain.StudyDoc{
		Topic:   "Chain rule",
		Started: start,
		Ended:   start.Add(time.Minute),
		Body:    "lesson body",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if rel != "study/2026-11-21-1405-chain-rule.md" {
		t.Fatalf("relative path = %q", rel)
	}
	docs, err := LoadStudy(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(docs) != 1 || docs[0].Topic != "Chain rule" || docs[0].Body != "lesson body\n" {
		t.Fatalf("docs = %+v", docs)
	}
	info, err := os.Stat(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
