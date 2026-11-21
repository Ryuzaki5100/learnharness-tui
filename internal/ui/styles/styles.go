// Package styles centralises the theme. Adaptive colours keep contrast on
// light and dark terminals without a second set of call sites.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	Accent  = lipgloss.AdaptiveColor{Light: "#6d28d9", Dark: "#c4a7f7"}
	Dim     = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#6b7280"}
	Faint   = lipgloss.AdaptiveColor{Light: "#9ca3af", Dark: "#4b5563"}
	Text    = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#e5e7eb"}
	Green   = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#6ee7b7"}
	Yellow  = lipgloss.AdaptiveColor{Light: "#a16207", Dark: "#fcd34d"}
	Cyan    = lipgloss.AdaptiveColor{Light: "#0e7490", Dark: "#67e8f9"}
	Magenta = lipgloss.AdaptiveColor{Light: "#a21caf", Dark: "#f0abfc"}
	Blue    = lipgloss.AdaptiveColor{Light: "#1d4ed8", Dark: "#93c5fd"}
	Red     = lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#fca5a5"}
)

// RoleColour maps a transcript entry role to its label colour.
func RoleColour(role string) lipgloss.AdaptiveColor {
	switch role {
	case "learner":
		return Green
	case "teacher":
		return Text
	case "question":
		return Yellow
	case "answer":
		return Cyan
	case "work":
		return Magenta
	case "artifact":
		return Blue
	default:
		return Dim
	}
}

// Header is the top application bar.
func Header(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

// TabActive is the selected tab.
func TabActive() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent).Padding(0, 1)
}

// TabIdle is an unselected tab.
func TabIdle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Dim).Padding(0, 1)
}

// Box frames a pane. Focus only changes the border colour.
func Box(width, height int, focused bool, title string) (lipgloss.Style, lipgloss.Style) {
	border := Dim
	if focused {
		border = Accent
	}
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Height(height)
	titleStyle := lipgloss.NewStyle().Foreground(border).Bold(true)
	if title == "" {
		return frame, titleStyle
	}
	return frame, titleStyle
}

// SelectedRow highlights the list cursor.
func SelectedRow(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.AdaptiveColor{Light: "#ede9fe", Dark: "#312e81"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#f5f3ff"})
}

// Row is a plain list row.
func Row(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

// Label styles a transcript role label.
func Label(role string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(RoleColour(role)).Bold(true)
}

// Muted renders secondary detail.
func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Dim)
}

// FaintStyle renders timestamps and hints.
func FaintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Faint)
}

// Error renders problems.
func Error() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Red)
}

// Hint renders empty-state guidance.
func Hint() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Faint).Italic(true)
}

// Status is the bottom mode line.
func Status(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width).Foreground(Dim)
}

// Title is a pane heading inside the frame.
func Title() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent)
}

// Wrap word-wraps plain text to width, preserving explicit newlines and
// hard-splitting tokens that are wider than the pane. It measures with
// runewidth so wide glyphs do not overflow the pane.
func Wrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		flush := func() {
			if line != "" {
				out = append(out, line)
				line = ""
			}
		}
		for _, word := range words {
			for _, chunk := range hardSplit(word, width) {
				switch {
				case line == "":
					line = chunk
				case runewidth.StringWidth(line)+1+runewidth.StringWidth(chunk) <= width:
					line += " " + chunk
				default:
					flush()
					line = chunk
				}
			}
		}
		flush()
	}
	return strings.Join(out, "\n")
}

func hardSplit(word string, width int) []string {
	if width <= 0 || runewidth.StringWidth(word) <= width {
		return []string{word}
	}
	var chunks []string
	var builder strings.Builder
	used := 0
	for _, r := range word {
		rw := runewidth.RuneWidth(r)
		if used+rw > width {
			chunks = append(chunks, builder.String())
			builder.Reset()
			used = 0
		}
		builder.WriteRune(r)
		used += rw
	}
	if builder.Len() > 0 {
		chunks = append(chunks, builder.String())
	}
	return chunks
}

// Truncate shortens a single line to width with an ellipsis.
func Truncate(text string, width int) string {
	if width <= 1 || runewidth.StringWidth(text) <= width {
		return text
	}
	var builder strings.Builder
	used := 0
	for _, r := range text {
		w := runewidth.RuneWidth(r)
		if used+w > width-1 {
			break
		}
		builder.WriteRune(r)
		used += w
	}
	return builder.String() + "…"
}
