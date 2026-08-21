package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/willgorman/teash/internal/app"
	"github.com/willgorman/teash/internal/teleport"
)

func testServerView(hostname, addr, os string, labels map[string]string) app.ServerView {
	return app.ServerView{
		Server: teleport.Server{
			Hostname: hostname,
			Addr:     addr,
			OS:       os,
		},
		AllLabels: labels,
	}
}

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

func TestColumnFilterColumnList_ClearFilter(t *testing.T) {
	cs := newColumnSelector([]string{"hostname", "os"})
	cs.selected = 1 // "os"
	m := Model{
		mode:        ModeColumnFilter,
		colSelector: cs,
		columnFilters: []app.ColumnFilter{
			{Column: "os", Values: []string{"linux"}},
		},
	}

	got, _ := m.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	gm := got.(Model)
	if len(gm.columnFilters) != 0 {
		t.Fatalf("expected os filter cleared by 'x', got %+v", gm.columnFilters)
	}
}

func TestColumnFilterValueChecklist_ArrowNavigation(t *testing.T) {
	servers := []app.ServerView{
		testServerView("web1", "10.0.0.1", "linux", nil),
		testServerView("web2", "10.0.0.2", "darwin", nil),
		testServerView("web3", "10.0.0.3", "windows", nil),
	}
	cs := newColumnSelector([]string{"os"}).enterValues(app.DistinctColumnValues(servers, "os")) // darwin, linux, windows
	m := Model{mode: ModeColumnFilterValue, servers: servers, colSelector: cs}

	got, _ := m.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyDown})
	gm := got.(Model)
	if gm.colSelector.valueCursor != 1 {
		t.Fatalf("valueCursor = %d, want 1 after down arrow", gm.colSelector.valueCursor)
	}

	got2, _ := gm.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	gm2 := got2.(Model)
	if gm2.colSelector.valueCursor != 2 {
		t.Fatalf("valueCursor = %d, want 2 after ctrl+n", gm2.colSelector.valueCursor)
	}

	got3, _ := gm2.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyUp})
	gm3 := got3.(Model)
	if gm3.colSelector.valueCursor != 1 {
		t.Fatalf("valueCursor = %d, want 1 after up arrow", gm3.colSelector.valueCursor)
	}
}

func TestColumnFilterValueChecklist_TypingNarrowsInsteadOfNavigating(t *testing.T) {
	servers := []app.ServerView{
		testServerView("web1", "10.0.0.1", "linux", nil),
		testServerView("web2", "10.0.0.2", "darwin", nil),
	}
	cs := newColumnSelector([]string{"os"}).enterValues(app.DistinctColumnValues(servers, "os")) // darwin, linux
	m := Model{mode: ModeColumnFilterValue, servers: servers, colSelector: cs}

	got, _ := m.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	gm := got.(Model)
	if gm.colSelector.valueCursor != 0 {
		t.Fatalf("expected 'j' not to move the cursor (it should narrow instead), got cursor=%d", gm.colSelector.valueCursor)
	}
	if gm.colSelector.input.Value() != "j" {
		t.Fatalf("expected 'j' to be typed into the narrow input, got %q", gm.colSelector.input.Value())
	}
}

func TestColumnFilterValueChecklist_ToggleTogglesFilter(t *testing.T) {
	servers := []app.ServerView{
		testServerView("web1", "10.0.0.1", "linux", nil),
		testServerView("web2", "10.0.0.2", "darwin", nil),
	}
	cs := newColumnSelector([]string{"os"}).enterValues(app.DistinctColumnValues(servers, "os")) // darwin, linux
	m := Model{mode: ModeColumnFilterValue, servers: servers, colSelector: cs}

	got, _ := m.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	gm := got.(Model)
	if len(gm.columnFilters) != 1 || gm.columnFilters[0].Column != "os" ||
		len(gm.columnFilters[0].Values) != 1 || gm.columnFilters[0].Values[0] != "darwin" {
		t.Fatalf("expected os=darwin filter added by space, got %+v", gm.columnFilters)
	}

	got2, _ := gm.handleColumnFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	gm2 := got2.(Model)
	if len(gm2.columnFilters) != 0 {
		t.Fatalf("expected filter removed after second toggle, got %+v", gm2.columnFilters)
	}
}

func TestColumnFilterKey_ClearsActiveSearch(t *testing.T) {
	m := Model{mode: ModeNormal, activeSearch: "foo"}

	got, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	gm := got.(Model)
	if gm.activeSearch != "" {
		t.Fatalf("expected activeSearch cleared when opening column filter, got %q", gm.activeSearch)
	}
	if gm.mode != ModeColumnFilter {
		t.Fatalf("expected mode ModeColumnFilter, got %v", gm.mode)
	}
}

func TestSearchKey_ClearsColumnFilters(t *testing.T) {
	m := Model{
		mode: ModeNormal,
		columnFilters: []app.ColumnFilter{
			{Column: "os", Values: []string{"linux"}},
		},
	}

	got, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	gm := got.(Model)
	if len(gm.columnFilters) != 0 {
		t.Fatalf("expected columnFilters cleared when starting a search, got %+v", gm.columnFilters)
	}
	if gm.mode != ModeSearch {
		t.Fatalf("expected mode ModeSearch, got %v", gm.mode)
	}
}

func TestFilteredServersSurviveResizeAndRefresh(t *testing.T) {
	servers := []app.ServerView{
		testServerView("web1", "10.0.0.1", "linux", map[string]string{"role": "web"}),
		testServerView("db1", "10.0.0.2", "linux", map[string]string{"role": "database"}),
	}
	m := Model{
		servers:   servers,
		labelKeys: []string{"role"},
		columnFilters: []app.ColumnFilter{
			{Column: "role", Values: []string{"web"}},
		},
		ready:  true,
		width:  120,
		height: 40,
	}
	m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	gm := updated.(Model)
	view := gm.table.View()
	if !strings.Contains(view, "web1") {
		t.Errorf("expected web1 to remain visible after resize, view=%s", view)
	}
	if strings.Contains(view, "db1") {
		t.Errorf("expected db1 to stay filtered out after resize, view=%s", view)
	}

	updated2, _ := gm.Update(serversLoadedMsg{servers: servers, labelKeys: []string{"role"}})
	gm2 := updated2.(Model)
	view2 := gm2.table.View()
	if !strings.Contains(view2, "web1") {
		t.Errorf("expected web1 to remain visible after refresh, view=%s", view2)
	}
	if strings.Contains(view2, "db1") {
		t.Errorf("expected db1 to stay filtered out after refresh, view=%s", view2)
	}
}
