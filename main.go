package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TODO: (willgorman)
//   - scroll bar
//   - initial load is slow and table draw is weird at first.  need a placeholder and loading indicator
//   - convert labels to columns
//   - column highlighting or scroll through ...
//   - tsh will automatically login if needed but then the output is not just json.
//     can i handle that so it still works?
//   - support different ssh usernames than the current user
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var tsh string

type model struct {
	table    table.Model
	teleport Teleport
	tshCmd   []string
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
		case "enter":
			m.tshCmd = []string{"tsh", "ssh", m.table.SelectedRow()[0]}
			return m, tea.Quit
		}
	case Nodes:
		nodes := Nodes(msg)
		// TODO: (willgorman) collect the set of all labels and make a column for each
		labelCols := map[string]int{}
		labelIdx := 2
		for _, n := range nodes {
			for l := range n.Labels {
				_, ok := labelCols[l]
				if ok {
					continue
				}
				labelIdx++
				labelCols[l] = labelIdx
			}
		}

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
		for _, n := range nodes {
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
	if len(m.tshCmd) > 0 {
		return ""
	}
	return baseStyle.Render(m.table.View()) + "\n"
}

func main() {
	tsh, _ = exec.LookPath("tsh")
	if tsh == "" {
		panic("teleport `tsh` command not found")
	}
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
	var err error
	if m, err = tea.NewProgram(model{table: t}).Run(); err != nil {
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
