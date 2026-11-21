package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionSummary is the list-row view of one chats/<id>.json record.
type SessionSummary struct {
	ID        string
	Title     string
	Kind      string // "teach" (default) or "practice"
	CreatedAt time.Time
	UpdatedAt time.Time
	Messages  int
	Path      string
}

// Chat is a full native session record (format v1).
type Chat struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	Version   int       `json:"version"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	Messages  []Message `json:"messages"`
}

// Message is one stored transcript entry. Role is user, assistant,
// toolResult or custom; content shapes differ per role.
type Message struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	Timestamp  int64           `json:"timestamp"`
	CustomType string          `json:"customType"`
	Content    json.RawMessage `json:"content"`
	Details    json.RawMessage `json:"details"`
	ToolName   string          `json:"toolName"`
	ToolCallID string          `json:"toolCallId"`
}

// Block is a content block inside an assistant message.
type Block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Output    json.RawMessage `json:"output"`
}

// Blocks decodes a message's content into blocks. A plain string content is
// returned as a single text block.
func (m Message) Blocks() []Block {
	if len(m.Content) == 0 {
		return nil
	}
	var text string
	if err := json.Unmarshal(m.Content, &text); err == nil {
		if text == "" {
			return nil
		}
		return []Block{{Type: "text", Text: text}}
	}
	var blocks []Block
	if err := json.Unmarshal(m.Content, &blocks); err == nil {
		return blocks
	}
	var single Block
	if err := json.Unmarshal(m.Content, &single); err == nil && single.Type != "" {
		return []Block{single}
	}
	return nil
}

// DetailsMap decodes a toolResult/custom details payload.
func (m Message) DetailsMap() map[string]any {
	value, err := decodeJSONValue(m.Details)
	if err != nil {
		return nil
	}
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return nil
}

// ArgumentsMap decodes a tool call's arguments, including the historical
// shape where the arguments were a JSON string rather than an object.
func (b Block) ArgumentsMap() map[string]any {
	return decodeObject(b.Arguments)
}

func decodeObject(raw json.RawMessage) map[string]any {
	value, err := decodeJSONValue(raw)
	if err != nil {
		return nil
	}
	object, _ := value.(map[string]any)
	return object
}

// decodeJSONValue accepts either a JSON value or a JSON string containing one.
func decodeJSONValue(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty payload")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil {
		if text, ok := value.(string); ok {
			var nested any
			if err := json.Unmarshal([]byte(text), &nested); err == nil {
				return nested, nil
			}
		}
		return value, nil
	}
	return nil, fmt.Errorf("invalid payload")
}

// LoadChat reads one native session record.
func LoadChat(dataDir, id string) (Chat, error) {
	path := filepath.Join(dataDir, "chats", id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Chat{}, fmt.Errorf("read session %s: %w", id, err)
	}
	var chat Chat
	if err := json.Unmarshal(raw, &chat); err != nil {
		return Chat{}, fmt.Errorf("parse session %s: %w", id, err)
	}
	if chat.ID == "" {
		chat.ID = id
	}
	if chat.Kind == "" {
		chat.Kind = "teach"
	}
	return chat, nil
}

// LoadSessionIndex lists chats/*.json, newest first. Unreadable records are
// skipped rather than failing the whole list; the number skipped is returned.
func LoadSessionIndex(dataDir string) ([]SessionSummary, int, error) {
	dir := filepath.Join(dataDir, "chats")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	var sessions []SessionSummary
	skipped := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		chat, err := LoadChat(dataDir, id)
		if err != nil {
			skipped++
			continue
		}
		sessions = append(sessions, SessionSummary{
			ID:        chat.ID,
			Title:     chat.Title,
			Kind:      chat.Kind,
			CreatedAt: parseTime(chat.CreatedAt),
			UpdatedAt: parseTime(chat.UpdatedAt),
			Messages:  len(chat.Messages),
			Path:      filepath.Join(dir, entry.Name()),
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	return sessions, skipped, nil
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
