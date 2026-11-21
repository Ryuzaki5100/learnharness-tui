// Package keys owns the single keymap source of truth. The footer and the
// ? overlay are generated from it; nothing is hand-written in prose.
package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap lists every binding once.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Top       key.Binding
	Bottom    key.Binding
	HalfUp    key.Binding
	HalfDown  key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Enter     key.Binding
	Back      key.Binding
	Search    key.Binding
	NextMatch key.Binding
	PrevMatch key.Binding
	TabLeft   key.Binding
	TabRight  key.Binding
	Sessions  key.Binding
	Graph     key.Binding
	Study     key.Binding
	NewStudy  key.Binding
	OpenStudy key.Binding
	Sync      key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding
}

// Default returns the application keymap.
func Default() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Left:      key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "pane left")),
		Right:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "pane right")),
		Top:       key.NewBinding(key.WithKeys("g"), key.WithHelp("gg", "top")),
		Bottom:    key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		HalfUp:    key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "half page up")),
		HalfDown:  key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "half page down")),
		PageUp:    key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "page up")),
		PageDown:  key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "page down")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open / start")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "normal")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		NextMatch: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		PrevMatch: key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "previous match")),
		TabLeft:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous tab")),
		TabRight:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		Sessions:  key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "sessions")),
		Graph:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "graph")),
		Study:     key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "study")),
		NewStudy:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "new study session")),
		OpenStudy: key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "interactive opencode")),
		Sync:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync now")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp is the one-line footer.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Down, k.Up, k.Right, k.Search, k.Sync, k.Help, k.Quit}
}

// FullHelp is the ? overlay, grouped by intent.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Down, k.Up, k.Top, k.Bottom, k.HalfDown, k.HalfUp, k.PageDown, k.PageUp},
		{k.Left, k.Right, k.Sessions, k.Graph, k.Study, k.Enter, k.Back},
		{k.Search, k.NextMatch, k.PrevMatch, k.NewStudy, k.OpenStudy, k.Sync, k.Refresh},
		{k.Help, k.Quit},
	}
}
