package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

func sized(t *testing.T, width, height int) Model {
	t.Helper()
	model := New(Options{DataDir: "/tmp/learnharness-test"})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: height})
	sizedModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T", updated)
	}
	return sizedModel
}

func keyRunes(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func TestLayoutAndView(t *testing.T) {
	model := sized(t, 120, 40)
	if !model.ready {
		t.Fatal("expected ready after WindowSizeMsg")
	}
	if model.layoutLeftWidth <= 0 || model.layoutRightWidth <= 0 || model.layoutBodyHeight <= 0 {
		t.Fatalf("layout not computed: %+v", model)
	}
	view := model.View()
	if !strings.Contains(view, "sessions") || !strings.Contains(view, "learnharness") {
		t.Fatalf("view missing chrome:\n%s", view)
	}
}

func TestMinimumSizeGuard(t *testing.T) {
	model := sized(t, 40, 10)
	if !strings.Contains(model.View(), "terminal too small") {
		t.Fatalf("expected size guard, got:\n%s", model.View())
	}
}

func TestTabKeysAndHelp(t *testing.T) {
	model := sized(t, 120, 40)

	updated, _ := model.Update(keyRunes("2"))
	model = updated.(Model)
	if model.tab != TabGraph {
		t.Fatalf("tab = %v, want graph", model.tab)
	}
	updated, _ = model.Update(keyRunes("3"))
	model = updated.(Model)
	if model.tab != TabStudy {
		t.Fatalf("tab = %v, want study", model.tab)
	}
	updated, _ = model.Update(keyRunes("?"))
	model = updated.(Model)
	if !model.helpVisible || !model.help.ShowAll {
		t.Fatal("help overlay did not open")
	}
	updated, _ = model.Update(keyRunes("esc"))
	model = updated.(Model)
	if model.helpVisible {
		t.Fatal("help overlay did not close")
	}
}

func TestQuitKey(t *testing.T) {
	model := sized(t, 120, 40)
	_, cmd := model.Update(keyRunes("q"))
	if cmd == nil {
		t.Fatal("q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q produced %T", cmd())
	}
}

func TestSessionsLoadedSelectsFirst(t *testing.T) {
	model := sized(t, 120, 40)
	updated, cmd := model.Update(sessionsLoadedMsg{sessions: []domain.SessionSummary{
		{ID: "chat_a", Title: "Alpha", Kind: "teach"},
		{ID: "chat_b", Title: "Beta", Kind: "practice"},
	}})
	model = updated.(Model)
	if len(model.sessions.filtered) != 2 {
		t.Fatalf("filtered = %d", len(model.sessions.filtered))
	}
	if cmd == nil {
		t.Fatal("expected a chat load command for the first session")
	}
	if _, ok := cmd().(chatLoadedMsg); !ok {
		t.Fatalf("expected chatLoadedMsg, got %T", cmd())
	}

	updated, cmd = model.Update(keyRunes("j"))
	model = updated.(Model)
	if summary, ok := model.sessions.current(); !ok || summary.ID != "chat_b" {
		t.Fatalf("cursor not moved: %+v", summary)
	}
	if cmd == nil {
		t.Fatal("moving the cursor should load the next session")
	}
}

func TestSessionsFilter(t *testing.T) {
	model := sized(t, 120, 40)
	updated, _ := model.Update(sessionsLoadedMsg{sessions: []domain.SessionSummary{
		{ID: "a", Title: "Alpha", Kind: "teach"},
		{ID: "b", Title: "Beta", Kind: "practice"},
	}})
	model = updated.(Model)

	updated, _ = model.Update(keyRunes("/"))
	model = updated.(Model)
	if model.mode != modeSearch {
		t.Fatal("search mode not entered")
	}
	model.sessions.setFilter("beta")
	if len(model.sessions.filtered) != 1 || model.sessions.filtered[0].ID != "b" {
		t.Fatalf("filter = %+v", model.sessions.filtered)
	}
	model.sessions.setFilter("")
	if len(model.sessions.filtered) != 2 {
		t.Fatalf("filter clear = %+v", model.sessions.filtered)
	}
}

func TestGraphDetailIncludesChains(t *testing.T) {
	graph := &domain.Graph{Props: map[string]*domain.Proposition{
		"n1": {ID: "n1", Name: "Outer rate"},
		"n2": {ID: "n2", Name: "Inner rate"},
		"n3": {ID: "n3", Name: "Chain rule"},
	}, Chains: []domain.Chain{{ID: "c1", Premises: []string{"n1", "n2"}, Conclusion: "n3"}}}
	graph.Order = []string{"n1", "n2", "n3"}

	model := sized(t, 120, 40)
	updated, _ := model.Update(graphLoadedMsg{graph: graph})
	model = updated.(Model)
	updated, _ = model.Update(keyRunes("2"))
	model = updated.(Model)

	proposition, ok := model.graph.current()
	if !ok || proposition.ID != "n3" {
		t.Fatalf("unexpected cursor: %+v", proposition)
	}
	detail := model.graph.detail.View()
	if !strings.Contains(detail, "Chain rule") || !strings.Contains(detail, "⇒") {
		t.Fatalf("detail missing chain context:\n%s", detail)
	}
}
