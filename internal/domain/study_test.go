package domain

import (
	"strings"
	"testing"
	"time"
)

func TestSlugAndFilename(t *testing.T) {
	if got := Slugify("Chain Rule: intuition!"); got != "chain-rule-intuition" {
		t.Fatalf("slug = %q", got)
	}
	if got := Slugify("   "); got != "study" {
		t.Fatalf("empty slug = %q", got)
	}
	start := time.Date(2026, 11, 21, 14, 5, 0, 0, time.UTC)
	if got := StudyFilename(start, "Chain Rule: intuition!"); got != "study/2026-11-21-1405-chain-rule-intuition.md" {
		t.Fatalf("filename = %q", got)
	}
}

func TestStudyRoundTrip(t *testing.T) {
	start := time.Date(2026, 11, 21, 14, 5, 0, 0, time.UTC)
	end := start.Add(12 * time.Minute)
	doc := StudyDoc{
		Topic:   "Chain rule",
		Command: "opencode run --format json",
		Started: start,
		Ended:   end,
		Body:    "## Lesson\n\nOuter times inner.",
	}
	raw := RenderStudy(doc)

	parsed, err := ParseStudy("/tmp/study/2026-11-21-1405-chain-rule.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Topic != doc.Topic || parsed.Command != doc.Command {
		t.Fatalf("metadata lost: %+v", parsed)
	}
	if !parsed.Started.Equal(start) || !parsed.Ended.Equal(end) {
		t.Fatalf("times lost: %v %v", parsed.Started, parsed.Ended)
	}
	if !strings.Contains(parsed.Body, "Outer times inner.") {
		t.Fatalf("body lost: %q", parsed.Body)
	}
	if parsed.LineCount() < 2 {
		t.Fatalf("line count = %d", parsed.LineCount())
	}
}

func TestStudyPromptNamesTheTopic(t *testing.T) {
	prompt := StudyPrompt("  the  chain\nrule ")
	if !strings.Contains(prompt, "the chain rule") {
		t.Fatalf("prompt = %q", prompt)
	}
	if strings.Contains(prompt, "\n") {
		t.Fatalf("prompt must stay a single line: %q", prompt)
	}
}

func TestParseStudyWithoutFrontmatter(t *testing.T) {
	doc, err := ParseStudy("/tmp/plain.md", []byte("just notes"))
	if err != nil || doc.Body != "just notes" {
		t.Fatalf("doc=%+v err=%v", doc, err)
	}
}
