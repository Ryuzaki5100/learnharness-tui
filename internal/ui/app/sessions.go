package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/styles"
)

type sessionsPane struct {
	all      []domain.SessionSummary
	filtered []domain.SessionSummary
	cursor   int
	offset   int
	width    int
	height   int
	err      error
	loading  bool
	skipped  int
	filter   string
}

func newSessionsPane() sessionsPane {
	return sessionsPane{loading: true}
}

func (p *sessionsPane) setSize(width, height int) {
	p.width, p.height = width, height
	p.ensureVisible()
}

func (p *sessionsPane) setSessions(list []domain.SessionSummary, skipped int) {
	p.all = list
	p.skipped = skipped
	p.loading = false
	p.err = nil
	p.applyFilter()
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
	p.ensureVisible()
}

func (p *sessionsPane) setFilter(query string) {
	p.filter = strings.ToLower(strings.TrimSpace(query))
	p.applyFilter()
	p.cursor = 0
	p.offset = 0
}

func (p *sessionsPane) applyFilter() {
	if p.filter == "" {
		p.filtered = p.all
		return
	}
	var out []domain.SessionSummary
	for _, session := range p.all {
		if strings.Contains(strings.ToLower(session.Title), p.filter) ||
			strings.Contains(strings.ToLower(session.Kind), p.filter) {
			out = append(out, session)
		}
	}
	p.filtered = out
}

func (p *sessionsPane) move(delta int) {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	p.ensureVisible()
}

func (p *sessionsPane) ensureVisible() {
	if p.height <= 0 {
		return
	}
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+p.height {
		p.offset = p.cursor - p.height + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p *sessionsPane) current() (domain.SessionSummary, bool) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return domain.SessionSummary{}, false
	}
	return p.filtered[p.cursor], true
}

func (p *sessionsPane) view() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	var lines []string
	switch {
	case p.err != nil:
		return padLines([]string{styles.Error().Render("error: " + p.err.Error())}, p.height)
	case p.loading:
		return padLines([]string{styles.Muted().Render("loading sessions…")}, p.height)
	case len(p.filtered) == 0:
		hint := "no sessions yet"
		if p.filter != "" {
			hint = "no session matches " + strconvQuote(p.filter)
		}
		return padLines([]string{styles.Hint().Render(hint)}, p.height)
	}

	for index := p.offset; index < len(p.filtered) && len(lines) < p.height; index++ {
		session := p.filtered[index]
		marker := "●"
		if session.Kind == "practice" {
			marker = "◆"
		}
		age := humanAge(session.UpdatedAt)
		available := p.width - runewidth.StringWidth(marker) - runewidth.StringWidth(age) - 2
		title := styles.Truncate(session.Title, max(4, available))
		row := marker + " " + title
		gap := p.width - runewidth.StringWidth(row) - runewidth.StringWidth(age)
		if gap > 0 {
			row += strings.Repeat(" ", gap) + age
		}
		if index == p.cursor {
			row = styles.SelectedRow(p.width).Render(row)
		}
		lines = append(lines, row)
	}
	if p.skipped > 0 && len(lines) < p.height {
		lines = append(lines, styles.FaintStyle().Render(fmt.Sprintf("%d unreadable record(s) skipped", p.skipped)))
	}
	return padLines(lines, p.height)
}

func humanAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	delta := time.Since(t)
	switch {
	case delta < time.Minute:
		return "now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh", int(delta.Hours()))
	case delta < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(delta.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func padLines(lines []string, height int) string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func strconvQuote(value string) string {
	return "\"" + value + "\""
}
