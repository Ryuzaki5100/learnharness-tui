package app

import (
	"time"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
)

// Messages are the only way I/O results reach Update.

type sessionsLoadedMsg struct {
	sessions []domain.SessionSummary
	skipped  int
	err      error
}

type chatLoadedMsg struct {
	id      string
	title   string
	entries []domain.Entry
	err     error
}

type graphLoadedMsg struct {
	graph *domain.Graph
	err   error
}

type studyIndexMsg struct {
	docs []domain.StudyDoc
	err  error
}

type studyDocMsg struct {
	doc domain.StudyDoc
	err error
}

type syncStatusMsg struct {
	status data.SyncStatus
}

type syncDoneMsg struct {
	action string
	out    string
	err    error
}

type studyStartedMsg struct {
	stream  *data.StudyStream
	topic   string
	started time.Time
	err     error
}

type studyLineMsg struct {
	line string
}

type studyEndedMsg struct {
	err   error
	ended time.Time
}

type studySavedMsg struct {
	rel string
	err error
}

type interactiveDoneMsg struct {
	err error
}
