package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"

	"github.com/willgorman/teash/internal/app"
)

// Mode represents the current TUI interaction mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeAddLabel
	ModeColumnFilter
	ModeColumnFilterValue
)

// Model is the top-level BubbleTea model for the TUI.
type Model struct {
	service       *app.Service
	table         table.Model
	servers       []app.ServerView
	labelKeys     []string
	mode          Mode
	width         int
	height        int
	err           error
	status        string
	ready         bool
	refreshing    bool
	labelInput    labelInput
	needLogin     bool
	searchInput   searchInput
	colSelector   columnSelector
	columnFilters []app.ColumnFilter // AND'd across columns, OR'd within a column
	activeSearch  string             // active '/' free-text query; mutually exclusive with columnFilters
}

// New creates a new TUI Model backed by the given service.
func New(service *app.Service) Model {
	return Model{
		service: service,
		mode:    ModeNormal,
	}
}

type serversLoadedMsg struct {
	servers   []app.ServerView
	labelKeys []string
}

type errMsg struct{ err error }

type statusMsg struct{ text string }

type loginDoneMsg struct{}

func (m Model) Init() tea.Cmd {
	if !m.service.IsLoggedIn() && m.service.ActiveCluster() != "" {
		m.needLogin = true
		cmd := m.service.LoginCmd()
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			return loginDoneMsg{}
		})
	}
	return m.loadServersCmd()
}

func (m Model) loadServersCmd() tea.Cmd {
	return func() tea.Msg {
		servers, err := m.service.LoadServers()
		if err != nil {
			return errMsg{err}
		}
		labelKeys := m.service.AllLabelKeys(servers)
		return serversLoadedMsg{servers: servers, labelKeys: labelKeys}
	}
}

type refreshDoneMsg struct{}

func (m Model) refreshServersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		servers, err := m.service.RefreshServers(ctx)
		if err != nil {
			return errMsg{err}
		}
		labelKeys := m.service.AllLabelKeys(servers)
		return serversLoadedMsg{servers: servers, labelKeys: labelKeys}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ready {
			m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		}
		return m, nil

	case serversLoadedMsg:
		m.servers = msg.servers
		m.labelKeys = msg.labelKeys
		m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		m.ready = true
		m.err = nil
		m.refreshing = false
		m.status = m.filterStatus()
		return m, nil

	case errMsg:
		m.err = msg.err
		m.refreshing = false
		m.status = ""
		return m, nil

	case statusMsg:
		m.status = msg.text
		return m, nil

	case loginDoneMsg:
		// After login, re-detect cluster and load servers
		return m, func() tea.Msg {
			if err := m.service.DetectCluster(context.Background()); err != nil {
				return errMsg{err}
			}
			servers, err := m.service.LoadServers()
			if err != nil {
				return errMsg{err}
			}
			labelKeys := m.service.AllLabelKeys(servers)
			return serversLoadedMsg{servers: servers, labelKeys: labelKeys}
		}

	case labelAddedMsg:
		if err := m.service.SetLabel(msg.hostname, msg.key, msg.value); err != nil {
			m.err = err
			m.mode = ModeNormal
			return m, nil
		}
		m.mode = ModeNormal
		m.status = fmt.Sprintf("Added %s:%s to %s", msg.key, msg.value, msg.hostname)
		return m, m.loadServersCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Pass to active inputs based on mode
	switch m.mode {
	case ModeAddLabel:
		var cmd tea.Cmd
		m.labelInput, cmd = m.labelInput.Update(msg)
		return m, cmd
	case ModeSearch:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	case ModeColumnFilterValue:
		var cmd tea.Cmd
		m.colSelector, cmd = m.colSelector.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always allow ctrl+c
	if isKey(msg, "ctrl+c") {
		return m, tea.Quit
	}

	// Handle mode-specific keys
	switch m.mode {
	case ModeAddLabel:
		return m.handleAddLabelKey(msg)
	case ModeSearch:
		return m.handleSearchKey(msg)
	case ModeColumnFilter, ModeColumnFilterValue:
		return m.handleColumnFilterKey(msg)
	}

	switch {
	case isKey(msg, KeyQuit):
		return m, tea.Quit
	case isKey(msg, KeyDown):
		m.table, _ = m.table.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case isKey(msg, KeyUp):
		m.table, _ = m.table.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case isKey(msg, KeyEnter):
		sv := m.SelectedServer()
		if sv != nil {
			login := m.service.DefaultLogin()
			cmd := m.service.SSHCmd(sv.Hostname, login)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				if err != nil {
					return errMsg{err}
				}
				return nil
			})
		}
		return m, nil
	case isKey(msg, KeyRefresh):
		if m.refreshing {
			return m, nil
		}
		m.refreshing = true
		m.status = "Refreshing..."
		return m, m.refreshServersCmd()
	case isKey(msg, KeyAddLabel):
		sv := m.SelectedServer()
		if sv != nil {
			m.mode = ModeAddLabel
			m.labelInput = newLabelInput(sv.Hostname)
			return m, m.labelInput.input.Focus()
		}
		return m, nil
	case isKey(msg, KeySearch):
		m.columnFilters = nil
		m.mode = ModeSearch
		m.searchInput = newSearchInput()
		m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		return m, m.searchInput.input.Focus()
	case isKey(msg, KeyColFilter):
		m.activeSearch = ""
		allCols := append([]string{colHostname, colIP, colOS}, m.labelKeys...)
		m.colSelector = newColumnSelector(allCols)
		m.mode = ModeColumnFilter
		m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		return m, nil
	}

	// Pass other keys to table for built-in handling
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyEscape):
		// Clear search and return to normal
		m.mode = ModeNormal
		m.activeSearch = ""
		m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		m.status = ""
		return m, nil
	case isKey(msg, KeyEnter):
		// Apply search
		m.mode = ModeNormal
		m.activeSearch = m.searchInput.Value()
		m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
		m.status = m.filterStatus()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) handleColumnFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == ModeColumnFilter {
		// Level 1: choosing a column
		switch {
		case isKey(msg, KeyEscape):
			m.mode = ModeNormal
			return m, nil
		case isKey(msg, KeyDown, KeyDownArrow, KeyCtrlNext):
			if m.colSelector.selected < len(m.colSelector.columns)-1 {
				m.colSelector.selected++
			}
			return m, nil
		case isKey(msg, KeyUp, KeyUpArrow, KeyCtrlPrev):
			if m.colSelector.selected > 0 {
				m.colSelector.selected--
			}
			return m, nil
		case isKey(msg, KeyClearFilter):
			col := m.colSelector.SelectedColumn()
			m.columnFilters = removeColumnFilter(m.columnFilters, col)
			m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
			m.status = m.filterStatus()
			return m, nil
		case isKey(msg, KeyEnter):
			col := m.colSelector.SelectedColumn()
			allValues := app.DistinctColumnValues(m.servers, col)
			m.colSelector = m.colSelector.enterValues(allValues)
			m.mode = ModeColumnFilterValue
			return m, m.colSelector.input.Focus()
		}
		return m, nil
	}

	// Level 2: value checklist for the chosen column
	switch {
	case isKey(msg, KeyEscape, KeyEnter):
		m.colSelector = m.colSelector.backToColumns()
		m.mode = ModeColumnFilter
		return m, nil
	case isKey(msg, KeyToggle):
		col := m.colSelector.SelectedColumn()
		val := m.colSelector.HighlightedValue()
		if val != "" {
			m.columnFilters = toggleColumnFilterValue(m.columnFilters, col, val)
			m.table = buildTable(m.visibleServers(), m.labelKeys, m.width, m.height)
			m.status = m.filterStatus()
		}
		return m, nil
	case isKey(msg, KeyDownArrow, KeyCtrlNext):
		if m.colSelector.valueCursor < len(m.colSelector.values)-1 {
			m.colSelector.valueCursor++
		}
		return m, nil
	case isKey(msg, KeyUpArrow, KeyCtrlPrev):
		if m.colSelector.valueCursor > 0 {
			m.colSelector.valueCursor--
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.colSelector, cmd = m.colSelector.Update(msg)
	return m, cmd
}

// removeColumnFilter returns filters with the entry for col dropped, if any.
func removeColumnFilter(filters []app.ColumnFilter, col string) []app.ColumnFilter {
	result := make([]app.ColumnFilter, 0, len(filters))
	for _, f := range filters {
		if f.Column != col {
			result = append(result, f)
		}
	}
	return result
}

// toggleColumnFilterValue adds val to col's OR set if absent, or removes it
// if present (dropping the column entirely once its set is empty).
func toggleColumnFilterValue(filters []app.ColumnFilter, col, val string) []app.ColumnFilter {
	for i, f := range filters {
		if f.Column != col {
			continue
		}
		for j, v := range f.Values {
			if v == val {
				f.Values = append(f.Values[:j], f.Values[j+1:]...)
				if len(f.Values) == 0 {
					return removeColumnFilter(filters, col)
				}
				filters[i] = f
				return filters
			}
		}
		f.Values = append(f.Values, val)
		filters[i] = f
		return filters
	}
	return append(filters, app.ColumnFilter{Column: col, Values: []string{val}})
}

// filterStatus renders a compact summary of whichever filter is active
// (column filters or free-text search), or "" if neither is active.
func (m Model) filterStatus() string {
	if len(m.columnFilters) > 0 {
		parts := make([]string, 0, len(m.columnFilters))
		for _, f := range m.columnFilters {
			parts = append(parts, fmt.Sprintf("%s=%s", f.Column, strings.Join(f.Values, ",")))
		}
		return fmt.Sprintf("Filters: %s (%d results)", strings.Join(parts, " AND "), len(m.visibleServers()))
	}
	if m.activeSearch != "" {
		return fmt.Sprintf("Filter: %q (%d results)", m.activeSearch, len(m.visibleServers()))
	}
	return ""
}

func (m Model) handleAddLabelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyEscape):
		m.mode = ModeNormal
		return m, nil
	case isKey(msg, KeyEnter):
		key, value, err := m.labelInput.Value()
		if err != nil {
			m.err = err
			return m, nil
		}
		hostname := m.labelInput.hostname
		m.mode = ModeNormal
		return m, func() tea.Msg {
			return labelAddedMsg{hostname: hostname, key: key, value: value}
		}
	}

	var cmd tea.Cmd
	m.labelInput, cmd = m.labelInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return "Loading servers...\n"
	}

	var s string

	if len(m.servers) == 0 {
		s += "No servers found. Press 'r' to refresh or check your tsh login.\n"
		s += m.renderStatusBar()
		return s
	}

	// Table
	s += m.table.View() + "\n"

	// Mode-specific overlays
	switch m.mode {
	case ModeAddLabel:
		s += m.labelInput.View() + "\n"
	case ModeSearch:
		s += m.searchInput.View() + "\n"
	case ModeColumnFilter, ModeColumnFilterValue:
		s += m.colSelector.View(m.columnFilters)
	default:
		s += m.renderStatusBar()
	}

	return s
}

// visibleServers returns m.servers narrowed by whichever filter is active.
// Column filters and free-text search are mutually exclusive.
func (m Model) visibleServers() []app.ServerView {
	if len(m.columnFilters) > 0 {
		return app.FilterServers(m.servers, m.columnFilters)
	}
	if m.activeSearch != "" {
		return app.FilterServersByText(m.servers, m.activeSearch)
	}
	return m.servers
}

// SelectedServer returns the currently highlighted server, if any.
func (m Model) SelectedServer() *app.ServerView {
	row := m.table.HighlightedRow()
	hostname, ok := row.Data[colHostname].(string)
	if !ok || hostname == "" {
		return nil
	}
	for i := range m.servers {
		if m.servers[i].Hostname == hostname {
			return &m.servers[i]
		}
	}
	return nil
}
