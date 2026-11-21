// Package data performs the file and process I/O that the UI requests
// through Bubble Tea commands. It never mutates LearnHarness-owned records.
package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

// LoadSessions lists the session index, newest first.
func LoadSessions(dataDir string) ([]domain.SessionSummary, int, error) {
	return domain.LoadSessionIndex(dataDir)
}

// LoadChat reads one session record.
func LoadChat(dataDir, id string) (domain.Chat, error) {
	return domain.LoadChat(dataDir, id)
}

// LoadGraph merges graph/nodes.jsonl and graph/chains.jsonl. Missing graph
// files are an empty graph, not an error: a fresh profile legitimately has
// none yet.
func LoadGraph(dataDir string) (*domain.Graph, error) {
	nodes, err := readIfExists(filepath.Join(dataDir, "graph", "nodes.jsonl"))
	if err != nil {
		return nil, err
	}
	chains, err := readIfExists(filepath.Join(dataDir, "graph", "chains.jsonl"))
	if err != nil {
		return nil, err
	}
	graph, err := domain.ParseGraph(nodes, chains)
	if err != nil {
		return nil, err
	}
	graph.Sources = loadSources(filepath.Join(dataDir, "graph", "sources.json"))
	return graph, nil
}

func loadSources(path string) map[string]domain.Source {
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]domain.Source{}
	}
	sources := map[string]domain.Source{}
	if err := json.Unmarshal(raw, &sources); err == nil {
		return sources
	}
	var list []domain.Source
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, source := range list {
			if source.ID != "" {
				sources[source.ID] = source
			}
		}
	}
	return sources
}

// LoadStudy lists study transcripts, newest filename first.
func LoadStudy(dataDir string) ([]domain.StudyDoc, error) {
	dir := filepath.Join(dataDir, "study")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list study sessions: %w", err)
	}
	var docs []domain.StudyDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		doc, err := domain.ParseStudy(path, raw)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	sort.Slice(docs, func(i, j int) bool {
		left, right := docs[i].Started, docs[j].Started
		if left.Equal(right) {
			return docs[i].Path > docs[j].Path
		}
		return left.After(right)
	})
	return docs, nil
}

// SaveStudy writes a study transcript atomically (temp + rename) so a crash
// never leaves a half-written note in the synced repository.
func SaveStudy(dataDir string, doc domain.StudyDoc) (string, error) {
	relative := domain.StudyFilename(doc.Started, doc.Topic)
	path := filepath.Join(dataDir, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create study directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".study-*.md")
	if err != nil {
		return "", fmt.Errorf("create study file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(domain.RenderStudy(doc)); err != nil {
		temp.Close()
		return "", fmt.Errorf("write study file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close study file: %w", err)
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		return "", fmt.Errorf("mode study file: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return "", fmt.Errorf("install study file: %w", err)
	}
	return relative, nil
}

func readIfExists(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		return raw, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, fmt.Errorf("read %s: %w", path, err)
}
