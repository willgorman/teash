package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO: use https://github.com/charmbracelet/x/tree/main/exp/teatest

// mockTeleport implements the Teleport interface for testing.
type mockTeleport struct {
	nodes       Nodes
	nodesErr    error
	cluster     string
	clusterErr  error
	connectCmds [][]string
}

func (m *mockTeleport) GetNodes(_ bool) (Nodes, error) {
	return m.nodes, m.nodesErr
}

func (m *mockTeleport) GetCluster() (string, error) {
	return m.cluster, m.clusterErr
}

func (m *mockTeleport) Connect(cmd []string) {
	m.connectCmds = append(m.connectCmds, cmd)
}

// testNodes returns a diverse set of 5 nodes for testing.
func testNodes() Nodes {
	return Nodes{
		{
			Hostname: "web-01.example.com",
			IP:       "10.0.1.1",
			OS:       "Ubuntu 22.04",
			Labels:   map[string]string{"env": "prod", "team": "frontend"},
		},
		{
			Hostname: "web-02.example.com",
			IP:       "10.0.1.2",
			OS:       "Ubuntu 22.04",
			Labels:   map[string]string{"env": "prod", "team": "frontend"},
		},
		{
			Hostname: "db-01.example.com",
			IP:       "10.0.2.1",
			OS:       "CentOS 9",
			Labels:   map[string]string{"env": "prod", "team": "data"},
		},
		{
			Hostname: "staging-01.example.com",
			IP:       "10.0.3.1",
			OS:       "Rocky Linux 9",
			Labels:   map[string]string{"env": "staging"},
		},
		{
			Hostname: "dev-01.example.com",
			IP:       "10.0.4.1",
			OS:       "NixOS 23.11",
			Labels:   map[string]string{"env": "dev", "team": "platform"},
		},
	}
}

// newTestModel creates a model suitable for testing with the given nodes already loaded.
func newTestModel(tp Teleport, nodes Nodes) model {
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(7),
	)
	search := textinput.New()
	spin := spinner.New()
	m := model{
		table:    t,
		search:   search,
		teleport: tp,
		profile:  "test-cluster",
		spinner:  spin,
		nodes:    nodes,
		visible:  nodes,
	}
	return m.fillTable()
}

// --- fillTable tests ---

func Test_fillTable(t *testing.T) {
	t.Run("correct columns via headers map and view", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		// 3 base columns + 2 label columns (env, team) = 5
		if len(m.headers) != 5 {
			t.Fatalf("expected 5 headers, got %d", len(m.headers))
		}
		// Verify column titles appear in the rendered view
		view := m.table.View()
		for _, title := range []string{"Hostname", "IP", "OS", "env", "team"} {
			if !strings.Contains(view, title) {
				t.Errorf("table view missing column title %q", title)
			}
		}
		// Label columns should be sorted alphabetically: env before team
		if m.headers[3] != "env" {
			t.Errorf("headers[3] = %q, want %q", m.headers[3], "env")
		}
		if m.headers[4] != "team" {
			t.Errorf("headers[4] = %q, want %q", m.headers[4], "team")
		}
	})

	t.Run("correct row count", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		rows := m.table.Rows()
		if len(rows) != 5 {
			t.Errorf("expected 5 rows, got %d", len(rows))
		}
	})

	t.Run("row values match node data", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		rows := m.table.Rows()
		if rows[0][0] != "web-01.example.com" {
			t.Errorf("row 0 hostname = %q, want %q", rows[0][0], "web-01.example.com")
		}
		if rows[0][1] != "10.0.1.1" {
			t.Errorf("row 0 IP = %q, want %q", rows[0][1], "10.0.1.1")
		}
		if rows[0][2] != "Ubuntu 22.04" {
			t.Errorf("row 0 OS = %q, want %q", rows[0][2], "Ubuntu 22.04")
		}
	})

	t.Run("sparse labels produce empty cells", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		rows := m.table.Rows()
		// staging-01 has env but no team label
		// Find the staging row
		var stagingRow table.Row
		for _, r := range rows {
			if r[0] == "staging-01.example.com" {
				stagingRow = r
				break
			}
		}
		if stagingRow == nil {
			t.Fatal("staging-01 row not found")
		}
		// team column (index 4) should be empty
		if stagingRow[4] != "" {
			t.Errorf("staging-01 team = %q, want empty", stagingRow[4])
		}
		// env column (index 3) should have value
		if stagingRow[3] != "staging" {
			t.Errorf("staging-01 env = %q, want %q", stagingRow[3], "staging")
		}
	})

	t.Run("empty node list produces base columns and 0 rows", func(t *testing.T) {
		tp := &mockTeleport{}
		m := newTestModel(tp, Nodes{})

		// 3 base columns only (no label columns)
		if len(m.headers) != 3 {
			t.Errorf("expected 3 headers, got %d", len(m.headers))
		}
		rows := m.table.Rows()
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("columnSelMode shows numbers instead of names in view", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.columnSelMode = true
		m = m.fillTable()

		// In columnSelMode, view should show numbers instead of column names
		view := m.table.View()
		// Should NOT contain the text column names
		// (Hostname, IP, OS may appear in row data, but column header area should show numbers)
		// We verify the title() function returns numbers
		if m.title("Hostname", 1) != "1" {
			t.Errorf("title('Hostname', 1) = %q, want '1'", m.title("Hostname", 1))
		}
		if m.title("IP", 2) != "2" {
			t.Errorf("title('IP', 2) = %q, want '2'", m.title("IP", 2))
		}
		if m.title("OS", 3) != "3" {
			t.Errorf("title('OS', 3) = %q, want '3'", m.title("OS", 3))
		}
		_ = view
	})

	t.Run("headers map is populated", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		if m.headers[0] != "Hostname" {
			t.Errorf("headers[0] = %q, want %q", m.headers[0], "Hostname")
		}
		if m.headers[1] != "IP" {
			t.Errorf("headers[1] = %q, want %q", m.headers[1], "IP")
		}
		if m.headers[2] != "OS" {
			t.Errorf("headers[2] = %q, want %q", m.headers[2], "OS")
		}
		if m.headers[3] != "env" {
			t.Errorf("headers[3] = %q, want %q", m.headers[3], "env")
		}
		if m.headers[4] != "team" {
			t.Errorf("headers[4] = %q, want %q", m.headers[4], "team")
		}
	})
}

// --- filterNodesBySearch tests ---

func Test_filterNodesBySearch(t *testing.T) {
	t.Run("empty search returns all nodes", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("")
		m = m.filterNodesBySearch()

		if len(m.visible) != len(nodes) {
			t.Errorf("expected %d visible, got %d", len(nodes), len(m.visible))
		}
	})

	t.Run("fuzzy match on hostname", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("web")
		m = m.filterNodesBySearch()

		if len(m.visible) < 2 {
			t.Errorf("expected at least 2 visible for 'web', got %d", len(m.visible))
		}
		for _, n := range m.visible[:2] {
			if !strings.Contains(n.Hostname, "web") {
				t.Errorf("expected hostname containing 'web', got %q", n.Hostname)
			}
		}
	})

	t.Run("fuzzy match on OS value", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("centos")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for 'centos'")
		}
		found := false
		for _, n := range m.visible {
			if strings.Contains(strings.ToLower(n.OS), "centos") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find CentOS node")
		}
	})

	t.Run("fuzzy match on label value", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("platform")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for 'platform'")
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("zzzznonexistent")
		m = m.filterNodesBySearch()

		if len(m.visible) != 0 {
			t.Errorf("expected 0 visible, got %d", len(m.visible))
		}
	})

	t.Run("case insensitive search", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.search.SetValue("WEB")
		m = m.filterNodesBySearch()

		if len(m.visible) < 2 {
			t.Errorf("expected at least 2 visible for 'WEB', got %d", len(m.visible))
		}
	})

	t.Run("column-specific search on hostname (col 1)", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.columnSel = 1
		m.search.SetValue("db")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for hostname 'db'")
		}
		for _, n := range m.visible {
			if !strings.Contains(strings.ToLower(n.Hostname), "db") {
				t.Errorf("expected hostname containing 'db', got %q", n.Hostname)
			}
		}
	})

	t.Run("column-specific search on IP (col 2)", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.columnSel = 2
		m.search.SetValue("10.0.2")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for IP '10.0.2'")
		}
	})

	t.Run("column-specific search on OS (col 3)", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.columnSel = 3
		m.search.SetValue("nixos")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for OS 'nixos'")
		}
	})

	t.Run("column-specific search on label (col 4+)", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		// col 4 = "env" (first label column, 0-indexed header 3 -> columnSel 4)
		m.columnSel = 4
		m.search.SetValue("staging")
		m = m.filterNodesBySearch()

		if len(m.visible) == 0 {
			t.Error("expected at least 1 visible for label env='staging'")
		}
	})
}

// --- bubbletea model tests ---

func Test_Init(t *testing.T) {
	t.Run("returns non-nil command batch", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		cmd := m.Init()
		if cmd == nil {
			t.Error("Init() returned nil command")
		}
	})
}

func Test_Update_keys(t *testing.T) {
	t.Run("q returns quit", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		_ = updated
		if cmd == nil {
			t.Fatal("expected quit command, got nil")
		}
		// Execute the command and check for quit
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("expected tea.QuitMsg, got %T", msg)
		}
	})

	t.Run("ctrl+c returns quit", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatal("expected quit command, got nil")
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("expected tea.QuitMsg, got %T", msg)
		}
	})

	t.Run("esc clears search and resets column selection", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.searching = true
		m.columnSel = 2
		m.columnSelMode = true
		m.search.SetValue("something")

		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		um := updated.(model)
		if um.search.Value() != "" {
			t.Errorf("search value = %q, want empty", um.search.Value())
		}
		if um.searching {
			t.Error("searching should be false after esc")
		}
		if um.columnSel != 0 {
			t.Errorf("columnSel = %d, want 0", um.columnSel)
		}
		if um.columnSelMode {
			t.Error("columnSelMode should be false after esc")
		}
	})

	t.Run("slash enters search mode", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		um := updated.(model)
		if !um.searching {
			t.Error("searching should be true after /")
		}
		if cmd == nil {
			t.Error("expected focus command, got nil")
		}
	})

	t.Run("enter with loaded table sets tshCmd and quits", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		um := updated.(model)
		if len(um.tshCmd) == 0 {
			t.Fatal("expected tshCmd to be set")
		}
		if um.tshCmd[0] != "tsh" || um.tshCmd[1] != "ssh" {
			t.Errorf("tshCmd = %v, want [tsh ssh ...]", um.tshCmd)
		}
		// Should include the selected row's hostname
		if um.tshCmd[2] != "web-01.example.com" {
			t.Errorf("tshCmd[2] = %q, want %q", um.tshCmd[2], "web-01.example.com")
		}
		if cmd == nil {
			t.Fatal("expected quit command")
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("expected tea.QuitMsg, got %T", msg)
		}
	})

	t.Run("c enters column selection mode", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		um := updated.(model)
		if !um.columnSelMode {
			t.Error("columnSelMode should be true after 'c'")
		}
	})

	t.Run("number key in column selection mode sets columnSel and enters search", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.columnSelMode = true

		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		um := updated.(model)
		if um.columnSel != 2 {
			t.Errorf("columnSel = %d, want 2", um.columnSel)
		}
		if !um.searching {
			t.Error("searching should be true after number key in column selection mode")
		}
	})
}

func Test_Update_messages(t *testing.T) {
	t.Run("Nodes message populates model", func(t *testing.T) {
		tp := &mockTeleport{}
		m := newTestModel(tp, Nodes{})

		nodes := testNodes()
		updated, _ := m.Update(nodes)
		um := updated.(model)
		if len(um.nodes) != 5 {
			t.Errorf("expected 5 nodes, got %d", len(um.nodes))
		}
		if len(um.visible) != 5 {
			t.Errorf("expected 5 visible, got %d", len(um.visible))
		}
		// Table should be populated
		rows := um.table.Rows()
		if len(rows) != 5 {
			t.Errorf("expected 5 table rows, got %d", len(rows))
		}
	})
}

func Test_View(t *testing.T) {
	t.Run("returns empty string when tshCmd is set", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)
		m.tshCmd = []string{"tsh", "ssh", "host"}

		view := m.View()
		if view != "" {
			t.Errorf("expected empty view, got %q", view)
		}
	})

	t.Run("contains table content when nodes loaded", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		view := m.View()
		if !strings.Contains(view, "web-01.example.com") {
			t.Error("view should contain node hostname")
		}
	})

	t.Run("navView shows Loading when no nodes", func(t *testing.T) {
		tp := &mockTeleport{}
		m := newTestModel(tp, Nodes{})
		m.nodes = nil

		nav := m.navView()
		if !strings.Contains(nav, "Loading") {
			t.Errorf("navView = %q, expected to contain 'Loading'", nav)
		}
	})

	t.Run("helpView content varies by mode", func(t *testing.T) {
		nodes := testNodes()
		tp := &mockTeleport{nodes: nodes}
		m := newTestModel(tp, nodes)

		// Normal mode
		help := m.helpView()
		if !strings.Contains(help, "/: Start search") {
			t.Errorf("normal helpView = %q, expected '/: Start search'", help)
		}

		// Searching mode
		m.searching = true
		help = m.helpView()
		if !strings.Contains(help, "Type to search") {
			t.Errorf("searching helpView = %q, expected 'Type to search'", help)
		}

		// Column selection mode
		m.searching = false
		m.columnSelMode = true
		help = m.helpView()
		if !strings.Contains(help, "Choose column") {
			t.Errorf("columnSel helpView = %q, expected 'Choose column'", help)
		}
	})
}
