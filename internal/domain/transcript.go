package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Entry roles used by the transcript renderer. They are presentation labels:
// "teacher" is model-authored prose, "learner" is the person's own words.
const (
	EntryLearner  = "learner"
	EntryTeacher  = "teacher"
	EntryQuestion = "question"
	EntryAnswer   = "answer"
	EntryWork     = "work"
	EntryArtifact = "artifact"
	EntryNote     = "note"
)

// Entry is one rendered transcript unit. It is a read-only projection of the
// stored record; it never invents learner state.
type Entry struct {
	Role            string
	Label           string
	Text            string
	Options         []string
	Confidence      *int
	Correct         *bool
	Thinking        string
	ReadingThinking string
	Extra           []string
	Time            time.Time
}

var questionTools = map[string]bool{
	"ask_user_question": true,
	"probe":             true,
	"quiz":              true,
	"recall":            true,
	"explain":           true,
}

var problemTools = map[string]bool{
	"practice": true,
	"problem":  true,
}

// TranscriptEntries projects a stored session into reading order. Teacher
// prose, questions, learner answers, practice work and delivered artifacts
// keep their own roles so they are never conflated.
func TranscriptEntries(chat Chat) []Entry {
	var entries []Entry
	for _, message := range chat.Messages {
		switch message.Role {
		case "user":
			text := strings.TrimSpace(joinText(message.Blocks()))
			if text == "" {
				continue
			}
			entries = append(entries, Entry{Role: EntryLearner, Label: "learner", Text: text, Time: messageTime(message)})
		case "assistant":
			for _, block := range message.Blocks() {
				switch block.Type {
				case "text":
					if text := strings.TrimSpace(block.Text); text != "" {
						entries = append(entries, Entry{Role: EntryTeacher, Label: "teacher", Text: text, Time: messageTime(message)})
					}
				case "toolCall":
					if entry, ok := toolCallEntry(block, messageTime(message)); ok {
						entries = append(entries, entry)
					}
				}
			}
		case "toolResult":
			entries = append(entries, toolResultEntries(message)...)
		case "custom":
			if entry, ok := customEntry(message); ok {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

func toolCallEntry(block Block, at time.Time) (Entry, bool) {
	args := block.ArgumentsMap()
	if args == nil {
		if strings.TrimSpace(block.Text) == "" {
			return Entry{}, false
		}
		return Entry{Role: EntryTeacher, Label: "teacher", Text: block.Text, Time: at}, true
	}
	switch {
	case questionTools[block.Name]:
		entry := Entry{Role: EntryQuestion, Label: questionLabel(block.Name), Text: firstString(args, "question", "questionText", "prompt", "text"), Time: at}
		entry.Options = optionLabels(args["options"])
		if entry.Text == "" {
			entry.Text = "(question text not stored)"
		}
		return entry, true
	case problemTools[block.Name]:
		text := firstString(args, "statement", "problem", "text")
		if text == "" {
			text = "(problem statement not stored)"
		}
		return Entry{Role: EntryWork, Label: "practice problem", Text: text, Time: at}, true
	case block.Name == "conclude":
		text := firstString(args, "outcome", "verdict", "text")
		if text == "" {
			text = "(no verdict text stored)"
		}
		return Entry{Role: EntryNote, Label: "practice conclude", Text: text, Time: at}, true
	default:
		if strings.TrimSpace(block.Text) != "" {
			return Entry{Role: EntryTeacher, Label: "teacher", Text: block.Text, Time: at}, true
		}
		return Entry{}, false
	}
}

func toolResultEntries(message Message) []Entry {
	details := message.DetailsMap()
	if details == nil {
		return nil
	}
	role := firstString(details, "kind", "tool")
	text := firstString(details, "questionText")
	var entries []Entry

	if text != "" || details["answers"] != nil || firstString(details, "text") != "" || firstString(details, "other") != "" {
		entry := Entry{Role: EntryAnswer, Label: answerLabel(role, message.ToolName), Time: messageTime(message)}
		labels := answerLabels(details["answers"])
		entry.Text = strings.Join(labels, ", ")
		if free := firstString(details, "text"); free != "" {
			if entry.Text != "" {
				entry.Text += " — "
			}
			entry.Text += free
		}
		if other := firstString(details, "other"); other != "" {
			if entry.Text != "" {
				entry.Text += " — "
			}
			entry.Text += "other: " + other
		}
		if truthy(details["dontKnow"]) {
			entry.Text = joinNonEmpty(entry.Text, "marked don't know")
		}
		if truthy(details["dismissed"]) {
			entry.Text = joinNonEmpty(entry.Text, "dismissed")
		}
		if entry.Text == "" {
			entry.Text = "(no learner response stored)"
		}
		entry.Confidence = intPtr(details["confidence"])
		entry.Correct = boolPtr(details["correct"])
		entry.Thinking = strings.TrimSpace(joinNonEmpty(firstString(details, "thinking"), firstString(details, "note")))
		entry.ReadingThinking = strings.TrimSpace(firstString(details, "readingThinking"))
		if committed := firstString(details, "committed"); committed != "" {
			entry.Extra = append(entry.Extra, "teacher's committed answer: "+committed)
		}
		if explanation := firstString(details, "explanation"); explanation != "" {
			entry.Extra = append(entry.Extra, "stored explanation: "+explanation)
		}
		if at := firstString(details, "answeredAt"); at != "" {
			if parsed := parseTime(at); !parsed.IsZero() {
				entry.Time = parsed
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

func customEntry(message Message) (Entry, bool) {
	details := message.DetailsMap()
	switch message.CustomType {
	case "practice-thinking":
		text := firstString(details, "text")
		if strings.TrimSpace(text) == "" {
			return Entry{}, false
		}
		return Entry{Role: EntryWork, Label: "learner practice submission", Text: text, Time: messageTime(message)}, true
	case "subagent-steer":
		kind := firstString(details, "kind")
		result := mapAt(details, "result")
		text := firstString(result, "brief", "notes", "title", "path")
		if text == "" {
			text = "(artifact stored without readable notes)"
		}
		label := "delivered artifact"
		if kind != "" {
			label += " (" + kind + ")"
		}
		return Entry{Role: EntryArtifact, Label: label, Text: text, Time: messageTime(message)}, true
	case "subagent-abort":
		return Entry{Role: EntryNote, Label: "background work aborted", Time: messageTime(message)}, true
	case "plan-block":
		return Entry{Role: EntryNote, Label: "plan snapshot", Text: "(session plan presentation, not the knowledge graph)", Time: messageTime(message)}, true
	case "graph-context":
		return Entry{Role: EntryNote, Label: "graph context supplied by the app", Time: messageTime(message)}, true
	case "practice-next", "problem-dismissed", "probe-handoff":
		return Entry{Role: EntryNote, Label: strings.ReplaceAll(message.CustomType, "-", " "), Time: messageTime(message)}, true
	case "compaction":
		return Entry{Role: EntryNote, Label: "context compaction", Text: "(management bookkeeping, not a learner statement)", Time: messageTime(message)}, true
	}
	return Entry{}, false
}

func joinText(blocks []Block) string {
	var parts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		case "image":
			parts = append(parts, "[image]")
		}
	}
	return strings.Join(parts, "\n")
}

func questionLabel(tool string) string {
	switch tool {
	case "probe", "quiz", "recall", "explain":
		return "check question (" + tool + ")"
	case "ask_user_question":
		return "ask"
	}
	return tool
}

func answerLabel(kind, tool string) string {
	if kind == "" {
		kind = tool
	}
	if kind == "" {
		return "learner response"
	}
	return "learner response (" + kind + ")"
}

func optionLabels(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	labels := make([]string, 0, len(list))
	for _, item := range list {
		switch value := item.(type) {
		case string:
			labels = append(labels, value)
		case map[string]any:
			labels = append(labels, firstString(value, "label", "value"))
		}
	}
	return labels
}

func answerLabels(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	labels := make([]string, 0, len(list))
	for _, item := range list {
		if object, ok := item.(map[string]any); ok {
			if label := firstString(object, "label", "value"); label != "" {
				labels = append(labels, label)
			}
		}
	}
	return labels
}

func firstString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mapAt(object map[string]any, key string) map[string]any {
	value, _ := object[key].(map[string]any)
	if value != nil {
		return value
	}
	if text, ok := object[key].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err == nil {
			return decoded
		}
	}
	return nil
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	}
	return false
}

func intPtr(value any) *int {
	switch typed := value.(type) {
	case float64:
		n := int(typed)
		return &n
	case int:
		n := typed
		return &n
	case json.Number:
		n, err := typed.Int64()
		if err != nil {
			return nil
		}
		v := int(n)
		return &v
	}
	return nil
}

func boolPtr(value any) *bool {
	if typed, ok := value.(bool); ok {
		return &typed
	}
	return nil
}

func joinNonEmpty(parts ...string) string {
	var kept []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, " · ")
}

func messageTime(message Message) time.Time {
	if message.Timestamp > 0 {
		return time.UnixMilli(message.Timestamp)
	}
	return time.Time{}
}

// SummaryLine is a one-line preview used in the sessions list.
func SummaryLine(chat Chat) string {
	for _, entry := range TranscriptEntries(chat) {
		if entry.Role == EntryTeacher {
			return truncateWords(entry.Text, 120)
		}
	}
	return ""
}

func truncateWords(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len([]rune(text)) <= limit {
		return text
	}
	return fmt.Sprintf("%s…", string([]rune(text)[:limit]))
}
