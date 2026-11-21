package domain

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Point is one provenance anchor in the graph.
type Point struct {
	ChatID    string `json:"chatId"`
	MessageID string `json:"messageId"`
}

// Proposition is a merged graph node assertion.
type Proposition struct {
	ID         string
	Name       string
	Statement  string
	Aliases    []string
	CreatedAt  string
	Provenance []Point
}

// Chain is a merged reasoning chain. All premises are used together.
type Chain struct {
	ID         string
	Premises   []string
	Conclusion string
	CreatedAt  string
	Provenance []Point
}

// Source is an imported-notes source record.
type Source struct {
	ID         string `json:"id"`
	FilePath   string `json:"filePath"`
	ImportedAt string `json:"importedAt"`
}

// Graph is the merged read model of graph/*.jsonl. It is not a mastery
// assessment: presence is not understanding.
type Graph struct {
	Props    map[string]*Proposition
	Order    []string
	Chains   []Chain
	Sources  map[string]Source
	Warnings []string
}

type rawProposition struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Statement  string   `json:"statement"`
	Aliases    []string `json:"aliases"`
	CreatedAt  string   `json:"createdAt"`
	Provenance []Point  `json:"provenance"`
}

type rawChain struct {
	ID         string   `json:"id"`
	Premises   []string `json:"premises"`
	Conclusion string   `json:"conclusion"`
	CreatedAt  string   `json:"createdAt"`
	Provenance []Point  `json:"provenance"`
}

// ParseGraph merges nodes.jsonl and chains.jsonl under the documented rules:
// the first line for a node wins its name/statement/date, aliases and
// provenance union, and chains merge on sorted premises plus conclusion.
func ParseGraph(nodes, chains []byte) (*Graph, error) {
	graph := &Graph{Props: map[string]*Proposition{}, Sources: map[string]Source{}}
	if err := parseNodes(nodes, graph); err != nil {
		return nil, err
	}
	if err := parseChains(chains, graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func parseNodes(data []byte, graph *Graph) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var raw rawProposition
		if err := json.Unmarshal([]byte(text), &raw); err != nil {
			graph.Warnings = append(graph.Warnings, fmt.Sprintf("nodes.jsonl line %d: %v", line, err))
			continue
		}
		if raw.ID == "" {
			graph.Warnings = append(graph.Warnings, fmt.Sprintf("nodes.jsonl line %d: missing id", line))
			continue
		}
		existing, ok := graph.Props[raw.ID]
		if !ok {
			graph.Props[raw.ID] = &Proposition{
				ID:         raw.ID,
				Name:       raw.Name,
				Statement:  raw.Statement,
				Aliases:    dedupe(raw.Aliases),
				CreatedAt:  raw.CreatedAt,
				Provenance: unionProvenance(nil, raw.Provenance),
			}
			graph.Order = append(graph.Order, raw.ID)
			continue
		}
		if strings.TrimSpace(raw.Statement) != "" {
			existing.Statement = joinStatement(existing.Statement, raw.Statement)
		}
		existing.Aliases = dedupe(append(existing.Aliases, raw.Aliases...))
		existing.Provenance = unionProvenance(existing.Provenance, raw.Provenance)
	}
	return scanner.Err()
}

func joinStatement(current, next string) string {
	switch {
	case strings.TrimSpace(current) == "":
		return next
	case current == next:
		return current
	default:
		return current + "\n" + next
	}
}

func parseChains(data []byte, graph *Graph) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	line := 0
	seen := map[string]int{}
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var raw rawChain
		if err := json.Unmarshal([]byte(text), &raw); err != nil {
			graph.Warnings = append(graph.Warnings, fmt.Sprintf("chains.jsonl line %d: %v", line, err))
			continue
		}
		if raw.Conclusion == "" || len(raw.Premises) == 0 {
			graph.Warnings = append(graph.Warnings, fmt.Sprintf("chains.jsonl line %d: incomplete chain", line))
			continue
		}
		key := chainKey(raw.Premises, raw.Conclusion)
		if index, ok := seen[key]; ok {
			graph.Chains[index].Provenance = unionProvenance(graph.Chains[index].Provenance, raw.Provenance)
			continue
		}
		seen[key] = len(graph.Chains)
		graph.Chains = append(graph.Chains, Chain{
			ID:         raw.ID,
			Premises:   append([]string(nil), raw.Premises...),
			Conclusion: raw.Conclusion,
			CreatedAt:  raw.CreatedAt,
			Provenance: unionProvenance(nil, raw.Provenance),
		})
	}
	return scanner.Err()
}

func chainKey(premises []string, conclusion string) string {
	sorted := append([]string(nil), premises...)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00") + "\x01" + conclusion
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func unionProvenance(current, next []Point) []Point {
	out := append([]Point(nil), current...)
	seen := map[Point]bool{}
	for _, point := range out {
		seen[point] = true
	}
	for _, point := range next {
		if point.ChatID == "" || seen[point] {
			continue
		}
		seen[point] = true
		out = append(out, point)
	}
	return out
}

// Name returns a proposition's display label, or a missing marker.
func (g *Graph) Name(id string) string {
	if proposition, ok := g.Props[id]; ok && proposition.Name != "" {
		return proposition.Name
	}
	return "<missing:" + id + ">"
}

// ChainText renders premises and conclusion with display names.
func (g *Graph) ChainText(chain Chain) string {
	var premises []string
	for _, id := range chain.Premises {
		premises = append(premises, g.Name(id))
	}
	return strings.Join(premises, " ∧ ") + " ⇒ " + g.Name(chain.Conclusion)
}

// Search returns propositions whose name, statement or aliases match the
// query. A bare term is a substring match; `*` is the only wildcard.
func (g *Graph) Search(query string) []*Proposition {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	parts := strings.Split(strings.ToLower(query), "*")
	var matches []*Proposition
	for _, id := range g.Order {
		proposition := g.Props[id]
		haystack := strings.ToLower(proposition.Name + "\n" + proposition.Statement + "\n" + strings.Join(proposition.Aliases, "\n"))
		if matchWildcard(haystack, parts) {
			matches = append(matches, proposition)
		}
	}
	return matches
}

func matchWildcard(haystack string, parts []string) bool {
	position := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		found := strings.Index(haystack[position:], part)
		if found < 0 {
			return false
		}
		position += found + len(part)
	}
	return true
}

// SortPropositions orders matches deterministically for the graph pane.
func SortPropositions(items []*Proposition) {
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}
