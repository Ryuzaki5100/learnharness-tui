package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/styles"
)

type graphPane struct {
	graph   *domain.Graph
	props   []*domain.Proposition
	cursor  int
	offset  int
	width   int
	height  int
	detail  viewport.Model
	filter  string
	err     error
	loading bool
}

func newGraphPane() graphPane {
	return graphPane{loading: true, detail: viewport.New(0, 0)}
}

func (p *graphPane) setSize(leftWidth, leftHeight, rightWidth, rightHeight int) {
	p.width, p.height = leftWidth, leftHeight
	p.detail.Width = rightWidth
	p.detail.Height = rightHeight
	if p.graph != nil {
		p.syncDetail()
	}
}

func (p *graphPane) setGraph(graph *domain.Graph) {
	p.graph = graph
	p.loading = false
	p.err = nil
	p.applyFilter()
	p.cursor = 0
	p.offset = 0
	p.syncDetail()
}

func (p *graphPane) setFilter(query string) {
	p.filter = strings.TrimSpace(query)
	p.applyFilter()
	p.cursor = 0
	p.offset = 0
	p.syncDetail()
}

func (p *graphPane) applyFilter() {
	p.props = nil
	if p.graph == nil {
		return
	}
	if p.filter == "" {
		for _, id := range p.graph.Order {
			p.props = append(p.props, p.graph.Props[id])
		}
	} else {
		p.props = p.graph.Search(p.filter)
	}
	sort.SliceStable(p.props, func(i, j int) bool {
		return strings.ToLower(p.props[i].Name) < strings.ToLower(p.props[j].Name)
	})
}

func (p *graphPane) count() int {
	if p.graph == nil {
		return 0
	}
	return len(p.graph.Order)
}

func (p *graphPane) move(delta int) {
	if len(p.props) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.props) {
		p.cursor = len(p.props) - 1
	}
	p.ensureVisible()
}

func (p *graphPane) ensureVisible() {
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

func (p *graphPane) current() (*domain.Proposition, bool) {
	if p.cursor < 0 || p.cursor >= len(p.props) {
		return nil, false
	}
	return p.props[p.cursor], true
}

func (p *graphPane) scrollDetail(delta int) {
	if delta > 0 {
		p.detail.LineDown(delta)
	} else {
		p.detail.LineUp(-delta)
	}
}

func (p *graphPane) syncDetail() {
	if p.graph == nil || p.width <= 0 {
		return
	}
	proposition, ok := p.current()
	if !ok {
		p.detail.SetContent(styles.Hint().Render("no proposition selected"))
		return
	}
	p.detail.SetContent(renderProposition(p.graph, proposition, p.detail.Width))
	p.detail.GotoTop()
}

func renderProposition(graph *domain.Graph, proposition *domain.Proposition, width int) string {
	var builder strings.Builder
	builder.WriteString(styles.Title().Render(proposition.Name))
	builder.WriteString("\n\n")
	if proposition.Statement != "" {
		builder.WriteString(styles.Wrap(proposition.Statement, width))
		builder.WriteString("\n\n")
	}
	if len(proposition.Aliases) > 0 {
		builder.WriteString(styles.FaintStyle().Render("aliases: " + strings.Join(proposition.Aliases, ", ")))
		builder.WriteString("\n")
	}
	builder.WriteString(styles.FaintStyle().Render(fmt.Sprintf("provenance: %d anchor(s)", len(proposition.Provenance))))
	builder.WriteString("\n\n")

	var chains []string
	for _, chain := range graph.Chains {
		for _, premise := range chain.Premises {
			if premise == proposition.ID {
				chains = append(chains, graph.ChainText(chain))
				break
			}
		}
		if chain.Conclusion == proposition.ID {
			chains = append(chains, graph.ChainText(chain))
		}
	}
	if len(chains) > 0 {
		builder.WriteString(styles.Title().Render("reasoning chains"))
		builder.WriteString("\n")
		for _, text := range dedupeStrings(chains) {
			builder.WriteString(styles.Wrap("• "+text, width))
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString(styles.Hint().Render("no chain references this proposition"))
	}
	return builder.String()
}

func (p *graphPane) listView() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	if p.err != nil {
		return padLines([]string{styles.Error().Render("error: " + p.err.Error())}, p.height)
	}
	if p.loading {
		return padLines([]string{styles.Muted().Render("loading graph…")}, p.height)
	}
	if p.graph == nil || len(p.graph.Order) == 0 {
		return padLines([]string{styles.Hint().Render("no propositions yet — teaching sessions populate the graph")}, p.height)
	}
	if len(p.props) == 0 {
		return padLines([]string{styles.Hint().Render("no proposition matches " + strconvQuote(p.filter))}, p.height)
	}
	var lines []string
	for index := p.offset; index < len(p.props) && len(lines) < p.height; index++ {
		row := styles.Truncate(p.props[index].Name, p.width)
		if index == p.cursor {
			row = styles.SelectedRow(p.width).Render(row)
		}
		lines = append(lines, row)
	}
	if len(p.graph.Warnings) > 0 && len(lines) < p.height {
		lines = append(lines, styles.FaintStyle().Render(fmt.Sprintf("%d warning(s) while reading graph", len(p.graph.Warnings))))
	}
	return padLines(lines, p.height)
}

func (p *graphPane) detailView() string {
	if p.detail.Width <= 0 || p.detail.Height <= 0 {
		return ""
	}
	if p.loading {
		return padLines([]string{styles.Muted().Render("loading graph…")}, p.detail.Height)
	}
	return p.detail.View()
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
