package domain

import (
	"strings"
	"testing"
)

func TestGraphMergeRules(t *testing.T) {
	nodes := strings.Join([]string{
		`{"id":"n1","name":"Derivative","statement":"A derivative is a limit of a difference quotient.","aliases":["rate of change"],"createdAt":"2026-11-01T00:00:00Z","provenance":[{"chatId":"chat_a","messageId":"m1"}]}`,
		`{"id":"n1","name":"Ignored later name","statement":"A derivative measures instantaneous rate.","aliases":["rate of change","slope"],"provenance":[{"chatId":"chat_a","messageId":"m1"},{"chatId":"chat_b","messageId":"m9"}]}`,
		`{"id":"n2","name":"Chain rule","statement":"","provenance":[{"chatId":"chat_b","messageId":"m2"}]}`,
		`{"id":"n3","name":"Composition","provenance":[]}`,
	}, "\n")
	chains := strings.Join([]string{
		`{"id":"c1","premises":["n1","n2"],"conclusion":"n3","createdAt":"2026-11-02T00:00:00Z","provenance":[{"chatId":"chat_b","messageId":"m3"}]}`,
		`{"id":"c2","premises":["n2","n1"],"conclusion":"n3","provenance":[{"chatId":"chat_c","messageId":"m4"}]}`,
		`{"id":"c3","premises":["n3"],"conclusion":"n_missing"}`,
		`{"id":"bad","premises":[],"conclusion":"n3"}`,
	}, "\n")

	graph, err := ParseGraph([]byte(nodes), []byte(chains))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	derivative := graph.Props["n1"]
	if derivative.Name != "Derivative" {
		t.Fatalf("later line replaced the name: %q", derivative.Name)
	}
	if !strings.Contains(derivative.Statement, "difference quotient") || !strings.Contains(derivative.Statement, "instantaneous rate") {
		t.Fatalf("statements not joined: %q", derivative.Statement)
	}
	if len(derivative.Aliases) != 2 {
		t.Fatalf("alias union wrong: %v", derivative.Aliases)
	}
	if len(derivative.Provenance) != 2 {
		t.Fatalf("provenance union wrong: %+v", derivative.Provenance)
	}

	if len(graph.Chains) != 2 {
		t.Fatalf("chains = %d, want 2 (sorted-premise duplicate merged): %+v", len(graph.Chains), graph.Chains)
	}
	if graph.Chains[0].ID != "c1" || len(graph.Chains[0].Premises) != 2 || graph.Chains[0].Premises[0] != "n1" {
		t.Fatalf("first assertion not preserved: %+v", graph.Chains[0])
	}
	if len(graph.Chains[0].Provenance) != 2 {
		t.Fatalf("chain provenance not unioned: %+v", graph.Chains[0].Provenance)
	}
	if graph.Name("n_missing") != "<missing:n_missing>" {
		t.Fatalf("missing endpoints must be reported: %q", graph.Name("n_missing"))
	}
	if len(graph.Warnings) != 1 {
		t.Fatalf("incomplete chain should warn once, got %v", graph.Warnings)
	}
}

func TestGraphSearchWildcard(t *testing.T) {
	graph := &Graph{Props: map[string]*Proposition{}}
	for _, item := range []struct{ id, name, statement string }{
		{"n1", "Chain rule", "compose outer and inner rates"},
		{"n2", "Product rule", "sum of two terms"},
		{"n3", "Integral", "area under a curve"},
	} {
		graph.Props[item.id] = &Proposition{ID: item.id, Name: item.name, Statement: item.statement}
		graph.Order = append(graph.Order, item.id)
	}

	if got := graph.Search("chain"); len(got) != 1 || got[0].ID != "n1" {
		t.Fatalf("substring search: %+v", got)
	}
	if got := graph.Search("*rule*"); len(got) != 2 {
		t.Fatalf("wildcard should match both rules: %+v", got)
	}
	if got := graph.Search("chain*rates"); len(got) != 1 {
		t.Fatalf("multi-part wildcard: %+v", got)
	}
	if got := graph.Search("nowhere"); len(got) != 0 {
		t.Fatalf("unexpected match: %+v", got)
	}
}

func TestChainTextUsesDisplayNames(t *testing.T) {
	graph := &Graph{Props: map[string]*Proposition{
		"a": {ID: "a", Name: "A"},
		"b": {ID: "b", Name: "B"},
		"c": {ID: "c", Name: "C"},
	}}
	text := graph.ChainText(Chain{Premises: []string{"a", "b"}, Conclusion: "c"})
	if text != "A ∧ B ⇒ C" {
		t.Fatalf("chain text = %q", text)
	}
}
