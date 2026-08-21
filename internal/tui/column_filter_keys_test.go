package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestColumnFilterSelector_NavigationKeys(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}},
		{"down arrow", tea.KeyMsg{Type: tea.KeyDown}},
		{"ctrl+n", tea.KeyMsg{Type: tea.KeyCtrlN}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				mode:        ModeColumnFilter,
				colSelector: newColumnSelector([]string{"Hostname", "IP", "OS"}),
			}
			got, _ := m.handleColumnFilterKey(tt.msg)
			gm := got.(Model)
			if gm.colSelector.selected != 1 {
				t.Fatalf("selected = %d, want 1 after %s", gm.colSelector.selected, tt.name)
			}
		})
	}
}

func TestColumnFilterSelector_ReverseNavigationKeys(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}},
		{"up arrow", tea.KeyMsg{Type: tea.KeyUp}},
		{"ctrl+p", tea.KeyMsg{Type: tea.KeyCtrlP}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newColumnSelector([]string{"Hostname", "IP", "OS"})
			cs.selected = 1
			m := Model{mode: ModeColumnFilter, colSelector: cs}
			got, _ := m.handleColumnFilterKey(tt.msg)
			gm := got.(Model)
			if gm.colSelector.selected != 0 {
				t.Fatalf("selected = %d, want 0 after %s", gm.colSelector.selected, tt.name)
			}
		})
	}
}
