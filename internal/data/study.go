package data

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// StudyStream is a running `opencode run` process. Lines carries rendered
// transcript lines; Err receives exactly one value when the process ends.
type StudyStream struct {
	Lines  <-chan string
	Err    <-chan error
	cancel context.CancelFunc
}

// Cancel stops the child process. Safe to call more than once.
func (s *StudyStream) Cancel() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

// OpenCodeAvailable reports whether the opencode CLI is on PATH.
func OpenCodeAvailable() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

// StartStudy runs `opencode run --format json <prompt>` and streams its
// events. The TUI owns the stream lifetime; cancelling the context kills the
// child.
func StartStudy(ctx context.Context, prompt string) (*StudyStream, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("opencode is not on PATH: install it (the dotfiles home-manager config provides it)")
	}
	childCtx, cancel := context.WithCancel(ctx)
	command := exec.CommandContext(childCtx, path, "run", "--format", "json", prompt)
	stdout, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stdout: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stderr: %w", err)
	}
	if err := command.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start opencode: %w", err)
	}

	lines := make(chan string, 64)
	errs := make(chan error, 1)
	stream := &StudyStream{Lines: lines, Err: errs, cancel: cancel}

	go func() {
		defer close(lines)
		stderrDone := make(chan struct{})
		go func() {
			defer close(stderrDone)
			drainStderr(stderr, lines)
		}()
		readEvents(stdout, lines)
		<-stderrDone
		waitErr := command.Wait()
		switch {
		case childCtx.Err() == context.Canceled:
			errs <- nil
		case waitErr != nil:
			errs <- fmt.Errorf("opencode exited: %w", waitErr)
		default:
			errs <- nil
		}
		close(errs)
	}()
	return stream, nil
}

func readEvents(reader io.Reader, lines chan<- string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if rendered, ok := renderEvent(scanner.Text()); ok {
			lines <- rendered
		}
	}
	if err := scanner.Err(); err != nil {
		lines <- "! stream error: " + err.Error()
	}
}

func drainStderr(reader io.Reader, lines chan<- string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 32*1024), 1024*1024)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			lines <- "! " + text
		}
	}
}

// renderEvent turns one `--format json` event into readable text. Unknown
// shapes fall back to the raw line so nothing is silently dropped.
func renderEvent(line string) (string, bool) {
	text := strings.TrimSpace(line)
	if text == "" {
		return "", false
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(text), &event); err != nil {
		return text, true
	}
	if kind, _ := event["type"].(string); kind == "error" {
		if message := firstText(event, "message"); message != "" {
			return "! " + message, true
		}
		return "! opencode reported an error", true
	}
	if content := extractText(event); content != "" {
		return content, true
	}
	if kind, _ := event["type"].(string); kind != "" {
		if tool := firstText(event, "tool"); tool != "" {
			return "· " + kind + ": " + tool, true
		}
		return "· " + kind, true
	}
	return "", false
}

// extractText walks the event for text-bearing fields, preferring the common
// streaming shapes and ignoring pure metadata.
func extractText(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"text", "delta", "content"} {
			if text, ok := typed[key].(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
		wrapped, _ := typed["part"].(map[string]any)
		if wrapped != nil {
			if text := extractText(wrapped); text != "" {
				return text
			}
		}
	case []any:
		var parts []string
		for _, item := range typed {
			if text := extractText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func firstText(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}
