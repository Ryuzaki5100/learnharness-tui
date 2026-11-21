package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/styles"
)

type transcriptPane struct {
	vp       viewport.Model
	entries  []domain.Entry
	title    string
	loading  bool
	err      error
	width    int
	height   int
	rendered int
}

func newTranscriptPane() transcriptPane {
	return transcriptPane{loading: true, vp: viewport.New(0, 0)}
}

func (p *transcriptPane) setSize(width, height int) {
	p.width, p.height = width, height
	p.vp.Width = width
	p.vp.Height = height
	if width != p.rendered {
		p.rendered = width
		p.rebuild()
	}
}

func (p *transcriptPane) innerWidth() int {
	return p.width
}

func (p *transcriptPane) setEntries(title string, entries []domain.Entry, width int) {
	p.title = title
	p.entries = entries
	p.loading = false
	p.err = nil
	p.rendered = width
	p.rebuild()
	p.vp.GotoTop()
}

func (p *transcriptPane) rebuild() {
	if p.width <= 0 {
		return
	}
	var blocks []string
	if p.err != nil {
		blocks = append(blocks, styles.Error().Render("error: "+p.err.Error()))
	}
	for _, entry := range p.entries {
		blocks = append(blocks, renderEntry(entry, p.width))
	}
	if len(blocks) == 0 {
		p.vp.SetContent(styles.Hint().Render("select a session to read it"))
		return
	}
	p.vp.SetContent(strings.Join(blocks, "\n\n"))
}

func renderEntry(entry domain.Entry, width int) string {
	var builder strings.Builder
	label := styles.Label(entry.Role).Render(entry.Label)
	if !entry.Time.IsZero() {
		label += " " + styles.FaintStyle().Render(entry.Time.Format("15:04"))
	}
	builder.WriteString(label)
	if entry.Confidence != nil {
		builder.WriteString("  " + styles.FaintStyle().Render(fmt.Sprintf("confidence %d%%", *entry.Confidence)))
	}
	if entry.Correct != nil {
		if *entry.Correct {
			builder.WriteString("  " + lipgloss.NewStyle().Foreground(styles.Green).Render("correct"))
		} else {
			builder.WriteString("  " + styles.Error().Render("incorrect"))
		}
	}
	builder.WriteString("\n")

	if entry.Text != "" {
		builder.WriteString(styles.Wrap(entry.Text, width))
		builder.WriteString("\n")
	}
	for index, option := range entry.Options {
		builder.WriteString(styles.Muted().Render(fmt.Sprintf("  [%d] %s", index+1, styles.Truncate(option, width-8))))
		builder.WriteString("\n")
	}
	if entry.Thinking != "" {
		builder.WriteString(styles.FaintStyle().Render("thinking: ") + styles.Wrap(entry.Thinking, width))
		builder.WriteString("\n")
	}
	if entry.ReadingThinking != "" {
		builder.WriteString(styles.FaintStyle().Render("reading: ") + styles.Wrap(entry.ReadingThinking, width))
		builder.WriteString("\n")
	}
	for _, extra := range entry.Extra {
		builder.WriteString(styles.FaintStyle().Render(styles.Wrap(extra, width)))
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (p *transcriptPane) view() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	if p.err != nil {
		return padLines([]string{styles.Error().Render("error: " + p.err.Error())}, p.height)
	}
	if len(p.entries) == 0 {
		hint := "no session selected"
		if p.title != "" {
			hint = "this session has no readable transcript"
		}
		return padLines([]string{styles.Hint().Render(hint)}, p.height)
	}
	return p.vp.View()
}
