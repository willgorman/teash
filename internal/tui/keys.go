package tui

import "github.com/charmbracelet/bubbletea"

// Key constants for the TUI.
const (
	KeyQuit       = "q"
	KeyUp         = "k"
	KeyDown       = "j"
	KeyEnter      = "enter"
	KeyRefresh    = "r"
	KeyAddLabel   = "a"
	KeySearch     = "/"
	KeyColFilter  = "c"
	KeyEscape     = "esc"
)

func isKey(msg tea.KeyMsg, keys ...string) bool {
	for _, k := range keys {
		if msg.String() == k {
			return true
		}
	}
	return false
}
