package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

// Every frame must fit the terminal exactly: one over-long line wraps and
// scrolls the header away.
func TestViewFitsTerminal(t *testing.T) {
	dir := "../../../testdata/golden"
	sessions, _, err := data.LoadSessions(dir)
	if err != nil || len(sessions) == 0 {
		t.Fatalf("golden sessions: %v", err)
	}
	chat, err := data.LoadChat(dir, sessions[0].ID)
	if err != nil {
		t.Fatalf("golden chat: %v", err)
	}
	entries := domain.TranscriptEntries(chat)

	for _, size := range []struct{ w, h int }{
		{80, 24},
		{96, 30},
		{120, 34},
		{160, 50},
	} {
		updated, _ := New(Options{DataDir: dir}).Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		model := updated.(Model)
		updated, _ = model.Update(chatLoadedMsg{title: chat.Title, entries: entries})
		model = updated.(Model)

		for _, helpOpen := range []bool{false, true} {
			view := model.View()
			if helpOpen {
				updated, _ = model.Update(keyRunes("?"))
				model = updated.(Model)
				view = model.View()
			}
			lines := strings.Split(view, "\n")
			if len(lines) > size.h {
				t.Fatalf("%dx%d: %d lines (help=%v)", size.w, size.h, len(lines), helpOpen)
			}
			for i, line := range lines {
				if width := lipgloss.Width(line); width > size.w {
					t.Fatalf("%dx%d: line %d is %d wide: %q", size.w, size.h, i, width, line)
				}
			}
			if helpOpen {
				updated, _ = model.Update(keyRunes("esc"))
				model = updated.(Model)
			}
		}
	}
}
