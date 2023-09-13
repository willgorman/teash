package main

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table    table.Model
	teleport Teleport
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case Nodes:
		nodes := Nodes(msg)
		// TODO: (willgorman) collect the set of all labels and make a column for each
		m.table.SetColumns([]table.Column{
			{Title: "Hostname", Width: 20},
			{Title: "IP", Width: 10},
		})
		rows := []table.Row{}
		for _, n := range nodes {
			rows = append(rows, table.Row{n.Hostname, n.IP})
		}
		m.table.SetRows(rows)
		return m, nil
	case error:
		panic(msg)
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func main() {
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
	if _, err := tea.NewProgram(model{table: t}).Run(); err != nil {
		panic(err)
	}
}
