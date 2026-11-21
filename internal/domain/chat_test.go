package domain

import (
	"testing"
	"time"
)

const sampleChat = `{
  "id": "chat_sample",
  "title": "Chain rule",
  "kind": "teach",
  "version": 1,
  "createdAt": "2026-11-20T10:00:00Z",
  "updatedAt": "2026-11-20T10:30:00Z",
  "messages": [
    {"id": "m1", "role": "user", "timestamp": 1763632800000, "content": "I want to understand the chain rule."},
    {"id": "m2", "role": "assistant", "timestamp": 1763632860000, "content": [
      {"type": "text", "text": "Let us start with a composition."},
      {"type": "toolCall", "id": "call_1", "name": "probe",
       "arguments": {"question": "Which claim follows?", "options": [{"label": "A one"}, {"label": "B two"}]}}
    ]},
    {"id": "m3", "role": "toolResult", "toolName": "probe", "toolCallId": "call_1", "timestamp": 1763632920000,
     "content": [{"type": "text", "text": "compact summary"}],
     "details": {"kind": "quiz", "answers": [{"label": "B two", "value": "b", "index": 2}],
                 "confidence": 70, "correct": true, "thinking": "because the inner rate multiplies",
                 "explanation": "The inner derivative factors out."}},
    {"id": "m4", "role": "custom", "customType": "practice-thinking", "timestamp": 1763632980000,
     "details": {"text": "attempt: 3x^2 * 2x"}},
    {"id": "m5", "role": "custom", "customType": "subagent-steer", "timestamp": 1763633040000,
     "details": {"kind": "visual", "result": {"notes": "arrows from inner to outer rate"}}}
  ]
}`

func TestLoadAndProjectChat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/chats/chat_sample.json", sampleChat)

	index, skipped, err := LoadSessionIndex(dir)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if skipped != 0 || len(index) != 1 {
		t.Fatalf("unexpected index: %+v skipped=%d", index, skipped)
	}
	if index[0].Title != "Chain rule" || index[0].Kind != "teach" || index[0].Messages != 5 {
		t.Fatalf("unexpected summary: %+v", index[0])
	}

	chat, err := LoadChat(dir, "chat_sample")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entries := TranscriptEntries(chat)

	roles := make([]string, 0, len(entries))
	for _, entry := range entries {
		roles = append(roles, entry.Role+":"+entry.Label)
	}
	want := []string{
		"learner:learner",
		"teacher:teacher",
		"question:check question (probe)",
		"answer:learner response (quiz)",
		"work:learner practice submission",
		"artifact:delivered artifact (visual)",
	}
	if len(roles) != len(want) {
		t.Fatalf("roles = %v, want %v", roles, want)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("entry %d = %q, want %q (all: %v)", i, roles[i], want[i], roles)
		}
	}

	question := entries[2]
	if question.Text != "Which claim follows?" || len(question.Options) != 2 || question.Options[1] != "B two" {
		t.Fatalf("question projection wrong: %+v", question)
	}
	answer := entries[3]
	if answer.Text != "B two" {
		t.Fatalf("answer text = %q", answer.Text)
	}
	if answer.Confidence == nil || *answer.Confidence != 70 {
		t.Fatalf("confidence = %v", answer.Confidence)
	}
	if answer.Correct == nil || !*answer.Correct {
		t.Fatalf("correct = %v", answer.Correct)
	}
	if answer.Thinking != "because the inner rate multiplies" {
		t.Fatalf("thinking = %q", answer.Thinking)
	}
	if entries[4].Text != "attempt: 3x^2 * 2x" {
		t.Fatalf("practice work = %q", entries[4].Text)
	}
	if entries[5].Text != "arrows from inner to outer rate" {
		t.Fatalf("artifact notes = %q", entries[5].Text)
	}
	if !entries[0].Time.Equal(time.UnixMilli(1763632800000)) {
		t.Fatalf("entry time = %v", entries[0].Time)
	}
}

func TestTranscriptKeepsPersistedQuestionSnapshot(t *testing.T) {
	// The answer's display order is authoritative; the authored option order
	// is not. The projection must not reorder the stored labels.
	chat := Chat{ID: "c", Messages: []Message{
		{ID: "1", Role: "toolResult", ToolName: "quiz", Details: []byte(`{"kind":"quiz","answers":[{"label":"B two","index":2},{"label":"A one","index":1}]}`)},
	}}
	entries := TranscriptEntries(chat)
	if len(entries) != 1 || entries[0].Text != "B two, A one" {
		t.Fatalf("unexpected projection: %+v", entries)
	}
}

func TestMissingDetailsDoesNotInventAnswers(t *testing.T) {
	chat := Chat{ID: "c", Messages: []Message{
		{ID: "1", Role: "toolResult", ToolName: "probe", Details: []byte(`{"kind":"quiz"}`)},
	}}
	if entries := TranscriptEntries(chat); len(entries) != 0 {
		t.Fatalf("expected no answer entry, got %+v", entries)
	}
}
