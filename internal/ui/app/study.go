package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/styles"
)

type studyPane struct {
	docs    []domain.StudyDoc
	cursor  int
	offset  int
	width   int
	height  int
	output  viewport.Model
	input   textinput.Model
	running bool
	stream  *data.StudyStream
	topic   string
	started time.Time
	ended   time.Time
	body    strings.Builder
	err     error
	saved   string
	showing string
}

func newStudyPane() studyPane {
	input := textinput.New()
	input.Placeholder = "what do you want to study?"
	input.Prompt = "› "
	input.CharLimit = 160
	return studyPane{output: viewport.New(0, 0), input: input}
}

func (p *studyPane) setSize(leftWidth, leftHeight, rightWidth, rightHeight int) {
	p.width, p.height = leftWidth, leftHeight
	p.output.Width = rightWidth
	p.output.Height = rightHeight
	p.input.Width = max(16, rightWidth-4)
	p.refreshOutput()
}

func (p *studyPane) setDocs(docs []domain.StudyDoc) {
	p.docs = docs
	if p.cursor >= len(docs) {
		p.cursor = max(0, len(docs)-1)
	}
	p.ensureVisible()
}

func (p *studyPane) move(delta int) {
	if len(p.docs) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.docs) {
		p.cursor = len(p.docs) - 1
	}
	p.ensureVisible()
}

func (p *studyPane) ensureVisible() {
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

func (p *studyPane) current() (domain.StudyDoc, bool) {
	if p.cursor < 0 || p.cursor >= len(p.docs) {
		return domain.StudyDoc{}, false
	}
	return p.docs[p.cursor], true
}

// reset prepares the output pane for a new session before opencode starts.
func (p *studyPane) reset(topic string) {
	p.topic = topic
	p.body.Reset()
	p.err = nil
	p.saved = ""
	p.showing = ""
	p.running = true
	p.started = time.Now()
	p.ended = time.Time{}
	p.refreshOutput()
}

func (p *studyPane) begin(stream *data.StudyStream, topic string, started time.Time) {
	p.stream = stream
	p.topic = topic
	p.started = started
	p.running = true
	p.refreshOutput()
}

func (p *studyPane) appendLine(line string) {
	if p.body.Len() > 0 {
		p.body.WriteString("\n")
	}
	p.body.WriteString(line)
	p.refreshOutput()
}

func (p *studyPane) finish(ended time.Time) {
	p.running = false
	p.ended = ended
	p.refreshOutput()
}

func (p *studyPane) document() domain.StudyDoc {
	return domain.StudyDoc{
		Topic:   p.topic,
		Command: "opencode run --format json",
		Started: p.started,
		Ended:   p.ended,
		Body:    p.body.String(),
	}
}

func (p *studyPane) showDoc(doc domain.StudyDoc) {
	p.showing = doc.Path
	p.output.SetContent(styles.Title().Render(doc.Topic) + "\n\n" + styles.Wrap(doc.Body, p.output.Width))
	p.output.GotoTop()
}

func (p *studyPane) scrollOutput(delta int) {
	if delta > 0 {
		p.output.LineDown(delta)
	} else {
		p.output.LineUp(-delta)
	}
}

func (p *studyPane) refreshOutput() {
	if p.showing != "" {
		return
	}
	text := strings.TrimSpace(p.body.String())
	if text == "" {
		hint := "press i to start a study session — opencode runs with the learnharness skill and the transcript is saved under study/"
		if p.running {
			hint = "starting opencode…"
		}
		p.output.SetContent(styles.Hint().Render(styles.Wrap(hint, p.output.Width)))
		return
	}
	header := styles.Title().Render(p.topic)
	if p.running {
		header += " " + styles.Muted().Render("running…")
	} else if p.saved != "" {
		header += " " + styles.Muted().Render("saved "+p.saved)
	}
	p.output.SetContent(header + "\n\n" + styles.Wrap(text, p.output.Width))
	p.output.GotoBottom()
}

func (p *studyPane) historyView() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	if p.err != nil {
		return padLines([]string{styles.Error().Render("error: " + p.err.Error())}, p.height)
	}
	if len(p.docs) == 0 {
		return padLines([]string{styles.Hint().Render("no study transcripts yet")}, p.height)
	}
	var lines []string
	for index := p.offset; index < len(p.docs) && len(lines) < p.height; index++ {
		doc := p.docs[index]
		when := doc.Started.Format("01-02 15:04")
		if doc.Started.IsZero() {
			when = "unknown"
		}
		label := when + "  " + doc.Topic
		row := styles.Truncate(label, p.width)
		if index == p.cursor {
			row = styles.SelectedRow(p.width).Render(row)
		}
		lines = append(lines, row)
	}
	return padLines(lines, p.height)
}

func (p *studyPane) outputView() string {
	if p.output.Width <= 0 || p.output.Height <= 0 {
		return ""
	}
	view := p.output.View()
	if p.running || p.input.Focused() {
		view = padLines(strings.Split(view, "\n"), p.output.Height-1) + "\n" + p.input.View()
	}
	return view
}
