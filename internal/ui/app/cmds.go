package app

import (
	"context"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

func loadSessionsCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		sessions, skipped, err := data.LoadSessions(dir)
		return sessionsLoadedMsg{sessions: sessions, skipped: skipped, err: err}
	}
}

func loadChatCmd(dir, id string) tea.Cmd {
	return func() tea.Msg {
		chat, err := data.LoadChat(dir, id)
		if err != nil {
			return chatLoadedMsg{id: id, err: err}
		}
		return chatLoadedMsg{id: id, title: chat.Title, entries: domain.TranscriptEntries(chat)}
	}
}

func loadGraphCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		graph, err := data.LoadGraph(dir)
		return graphLoadedMsg{graph: graph, err: err}
	}
}

func loadStudyIndexCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		docs, err := data.LoadStudy(dir)
		return studyIndexMsg{docs: docs, err: err}
	}
}

func loadStudyDocCmd(path string) tea.Cmd {
	return func() tea.Msg {
		raw, err := os.ReadFile(path)
		if err != nil {
			return studyDocMsg{err: err}
		}
		doc, err := domain.ParseStudy(path, raw)
		return studyDocMsg{doc: doc, err: err}
	}
}

func syncStatusCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		return syncStatusMsg{status: data.ReadSyncStatus(dir)}
	}
}

func syncCmd(dir, action string) tea.Cmd {
	return func() tea.Msg {
		out, err := data.RunSync(dir, action)
		return syncDoneMsg{action: action, out: out, err: err}
	}
}

func saveStudyCmd(dir string, doc domain.StudyDoc) tea.Cmd {
	return func() tea.Msg {
		rel, err := data.SaveStudy(dir, doc)
		return studySavedMsg{rel: rel, err: err}
	}
}

func startStudyCmd(topic string) tea.Cmd {
	return func() tea.Msg {
		stream, err := data.StartStudy(context.Background(), domain.StudyPrompt(topic))
		return studyStartedMsg{stream: stream, topic: topic, started: time.Now(), err: err}
	}
}

func waitStudyLineCmd(stream *data.StudyStream) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-stream.Lines
		if !ok {
			err := <-stream.Err
			return studyEndedMsg{err: err, ended: time.Now()}
		}
		return studyLineMsg{line: line}
	}
}

func interactiveOpenCodeCmd() tea.Cmd {
	command := exec.Command("opencode")
	return tea.ExecProcess(command, func(err error) tea.Msg {
		return interactiveDoneMsg{err: err}
	})
}
