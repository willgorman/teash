package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/exp/maps"
)

// TODO: (willgorman)
//   - scroll bar
//   - initial load is slow and table draw is weird at first.  need a placeholder and loading indicator
//   - convert labels to columns
//   - column highlighting or scroll through ...
//   - tsh will automatically login if needed but then the output is not just json.
//     can i handle that so it still works?
//   - support different ssh usernames than the current user
//   - remove / from search input
//   - rank the rows by best match?
//   - highlight matching characters?
//   - select column to search only that column?
//   - altscreen with key help
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var tsh string

type model struct {
	table     table.Model
	search    textinput.Model
	teleport  Teleport
	tshCmd    []string
	nodes     Nodes
	visible   Nodes
	searching bool
}

// Init is the first function that will be called. It returns an optional
// initial command. To not perform an initial command return nil.
func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		nodes, err := m.teleport.GetNodes(true)
		if err != nil {
			return err
		}
		return nodes
	}
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.searching {
		m.search.Focus()
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.search.Focused() {
				m.search.Blur()
				m.search.SetValue("")
				m.searching = false
			}
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q":
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		case "/":
			m.searching = true
		case "enter":
			m.tshCmd = []string{"tsh", "ssh", m.table.SelectedRow()[0]}
			return m, tea.Quit
		}
	case Nodes:
		m.nodes = Nodes(msg)
		m.visible = m.nodes
		return m.fillTable(), nil
	case error:
		panic(msg)
	}
	m.search, _ = m.search.Update(msg)
	m = m.filterNodesBySearch()
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m model) View() string {
	if len(m.tshCmd) > 0 {
		return ""
	}
	return baseStyle.Render(m.table.View()) + "\n" + m.search.View() + "\n"
}

func (m model) fillTable() model {
	labelCols := map[string]int{}
	labelIdx := 2
	for _, n := range m.nodes {
		for l := range n.Labels {
			_, ok := labelCols[l]
			if ok {
				continue
			}
			labelIdx++
			labelCols[l] = labelIdx
		}
	}
	// TODO: (willgorman) sort columns by name for consistent order

	columns := make([]table.Column, len(labelCols)+3)
	columns[0] = table.Column{Title: "Hostname", Width: 30}
	columns[1] = table.Column{Title: "IP", Width: 16}
	columns[2] = table.Column{Title: "OS", Width: 30}
	for l, v := range labelCols {
		columns[v] = table.Column{Title: l, Width: 15}
	}
	// TODO: (willgorman) calculate widths by largest value in the column.  but what's the
	// ideal max width?
	m.table.SetColumns(columns)
	rows := []table.Row{}
	for _, n := range m.visible {
		row := make(table.Row, len(labelCols)+3)
		row[0] = n.Hostname
		row[1] = n.IP
		row[2] = n.OS
		for l, v := range n.Labels {
			i, ok := labelCols[l]
			if !ok {
				continue
			}
			row[i] = v
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
	return m
}

func (m model) filterNodesBySearch() model {
	if m.search.Value() == "" {
		return m
	}
	m.visible = nil
	txt2node := map[string]Node{}
	for _, n := range m.nodes {
		allText := n.Hostname + " " + n.IP + " " + n.OS
		for _, v := range n.Labels {
			allText = allText + " " + v
		}
		txt2node[allText] = n
	}
	ranks := fuzzy.RankFind(m.search.Value(), maps.Keys(txt2node))
	for _, rank := range ranks {
		m.visible = append(m.visible, txt2node[rank.Target])
	}

	return m.fillTable()
}

func main() {
	var err error
	tsh, _ = exec.LookPath("tsh")
	if tsh == "" {
		panic("teleport `tsh` command not found")
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(7),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	var m tea.Model

	search := textinput.New()
	if m, err = tea.NewProgram(model{table: t, search: search}).Run(); err != nil {
		panic(err)
	}

	model := m.(model)
	if len(model.tshCmd) == 0 {
		return
	}

	err = syscall.Exec(tsh, model.tshCmd, os.Environ())
	if err != nil {
		panic(err)
	}
}
