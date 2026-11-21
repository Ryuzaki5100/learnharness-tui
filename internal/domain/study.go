package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// StudyDoc is one opencode-backed study transcript stored under study/.
// These are TUI-authored notes, not native LearnHarness teaching records.
type StudyDoc struct {
	Topic   string
	Model   string
	Started time.Time
	Ended   time.Time
	Command string
	Body    string
	Path    string
}

// StudyFilename returns study/YYYY-MM-DD-HHMM-<slug>.md.
func StudyFilename(start time.Time, topic string) string {
	return fmt.Sprintf("study/%s-%s.md", start.Format("2006-01-02-1504"), slug(topic))
}

// Slugify lowercases and hyphenates a topic for filenames.
func Slugify(topic string) string { return slug(topic) }

func slug(topic string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(topic)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		out = "study"
	}
	if len(out) > 60 {
		out = strings.TrimRight(out[:60], "-")
	}
	return out
}

// RenderStudy serialises a study document with frontmatter the TUI can read
// back without a YAML dependency.
func RenderStudy(doc StudyDoc) []byte {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("topic: " + singleLine(doc.Topic) + "\n")
	if doc.Model != "" {
		builder.WriteString("model: " + singleLine(doc.Model) + "\n")
	}
	builder.WriteString("started: " + doc.Started.Format(time.RFC3339) + "\n")
	if !doc.Ended.IsZero() {
		builder.WriteString("ended: " + doc.Ended.Format(time.RFC3339) + "\n")
	}
	if doc.Command != "" {
		builder.WriteString("command: " + singleLine(doc.Command) + "\n")
	}
	builder.WriteString("---\n\n")
	builder.WriteString("# " + singleLine(doc.Topic) + "\n\n")
	builder.WriteString(strings.TrimSpace(doc.Body))
	builder.WriteString("\n")
	return []byte(builder.String())
}

// ParseStudy reads a study document written by RenderStudy. Unknown keys are
// ignored; the body is everything after the closing frontmatter fence.
func ParseStudy(path string, data []byte) (StudyDoc, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return StudyDoc{Path: path, Body: text}, nil
	}
	rest := strings.TrimPrefix(text, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return StudyDoc{}, fmt.Errorf("%s: unterminated frontmatter", path)
	}
	header, body := rest[:end], rest[end+5:]
	doc := StudyDoc{Path: path}
	for _, line := range strings.Split(header, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "topic":
			doc.Topic = value
		case "model":
			doc.Model = value
		case "command":
			doc.Command = value
		case "started":
			doc.Started = parseLooseTime(value)
		case "ended":
			doc.Ended = parseLooseTime(value)
		}
	}
	body = strings.TrimPrefix(body, "\n")
	// The file carries a human-readable H1 for editors; the TUI renders the
	// topic itself, so strip the duplicate heading on read.
	if doc.Topic != "" {
		body = strings.TrimPrefix(body, "# "+doc.Topic+"\n\n")
	}
	doc.Body = body
	if doc.Topic == "" {
		doc.Topic = strings.TrimSuffix(filepathBase(path), ".md")
	}
	return doc, nil
}

func parseLooseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func filepathBase(path string) string {
	if index := strings.LastIndexByte(path, '/'); index >= 0 {
		return path[index+1:]
	}
	return path
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// LineCount counts non-empty body lines for list rows.
func (d StudyDoc) LineCount() int {
	count := 0
	for _, line := range strings.Split(d.Body, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// StudyPrompt is the prompt the TUI sends to `opencode run`.
func StudyPrompt(topic string) string {
	return "Use the learnharness skill to read my learning record and adapt to what I already understand. " +
		"Run a focused study session on: " + singleLine(topic) + ". " +
		"Start from one concrete claim, check my understanding with a question, then explain and connect it to what I know. " +
		"Keep the session self-contained and end with the single most important idea to remember."
}

// IntOr renders an optional int for list rows.
func IntOr(value *int, fallback string) string {
	if value == nil {
		return fallback
	}
	return strconv.Itoa(*value)
}
