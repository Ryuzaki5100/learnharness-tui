// Command lhtui reads a LearnHarness learning record and, optionally, runs
// opencode study sessions that are saved into the synced data repository.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/domain"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/app"
)

var version = "dev"

func main() {
	var (
		dataDirFlag string
		noSync      bool
		showVersion bool
	)
	flag.StringVar(&dataDirFlag, "data-dir", "", "LearnHarness data directory (default: $LH_DATA_DIR or the OS default)")
	flag.BoolVar(&noSync, "no-sync", false, "do not run lh-sync pull/push around this session")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("lhtui", version)
		return
	}

	dir, err := domain.ResolveDataDir(dataDirFlag)
	if err != nil {
		fail(err.Error())
	}

	if !noSync && data.SyncAvailable() {
		out, err := data.RunSync(dir, "pull")
		if err != nil {
			fail("refusing to start: " + err.Error())
		}
		if out != "" {
			fmt.Fprintln(os.Stderr, "lh-sync:", firstLine(out))
		}
	}

	if !domain.DataDirExists(dir) {
		fail(fmt.Sprintf("no LearnHarness data at %s\nrun `lh-sync init` or pass --data-dir", dir))
	}

	program := tea.NewProgram(app.New(app.Options{
		DataDir:     dir,
		SyncEnabled: !noSync,
	}), tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fail(err.Error())
	}

	if !noSync && data.SyncAvailable() {
		if out, err := data.RunSync(dir, "push"); err != nil {
			fmt.Fprintln(os.Stderr, "lh-sync:", err.Error())
		} else if out != "" {
			fmt.Fprintln(os.Stderr, "lh-sync:", firstLine(out))
		}
	}
}

func firstLine(text string) string {
	for index, r := range text {
		if r == '\n' {
			return text[:index]
		}
	}
	return text
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "lhtui:", message)
	os.Exit(1)
}
