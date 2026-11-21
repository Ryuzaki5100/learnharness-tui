package data

import (
	"path/filepath"
	"testing"
)

// The golden fixtures are the LearnHarness project's own test records
// (AGPL-3.0), copied unchanged so these readers are exercised against real
// stored shapes rather than only hand-written ones.
func TestGoldenRecordParity(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "golden")

	sessions, skipped, err := LoadSessions(dir)
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("skipped %d golden records", skipped)
	}
	if len(sessions) == 0 {
		t.Fatal("no golden sessions loaded")
	}
	for _, session := range sessions {
		chat, err := LoadChat(dir, session.ID)
		if err != nil {
			t.Fatalf("%s: %v", session.ID, err)
		}
		if len(chat.Messages) == 0 {
			t.Fatalf("%s has no messages", session.ID)
		}
		if len(chat.Messages) != session.Messages {
			t.Fatalf("%s message count %d != summary %d", session.ID, len(chat.Messages), session.Messages)
		}
	}

	graph, err := LoadGraph(dir)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if len(graph.Order) == 0 || len(graph.Chains) == 0 {
		t.Fatalf("golden graph did not merge: props=%d chains=%d", len(graph.Order), len(graph.Chains))
	}
	if len(graph.Warnings) > 0 {
		t.Fatalf("golden graph produced warnings: %v", graph.Warnings)
	}
	if len(graph.Sources) == 0 {
		t.Fatal("golden sources not loaded")
	}
}
