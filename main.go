package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sanity-io/litter"
	"golang.org/x/exp/maps"
)

// TODO: (willgorman)
//   - fix columns selection for labels
//   - ranking still seems weird. levenstein distance can be the same for two results
//     where one has a prefix of the search term and one does not but the one without
//     the prefix may be shown first...
//   - remove / from search input
//   - rank the rows by best match?
//
// FIXME:
// - if the current table cursor is > than the number of rows that are left after applying a search then the table will appear empty until moving the cursor
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var tsh string

type model struct {
	table         table.Model
	search        textinput.Model
	teleport      Teleport
	tshCmd        []string
	nodes         Nodes
	visible       Nodes
	searching     bool
	columnSelMode bool
	columnSel     int
	headers       map[int]string
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
			}
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
			m.search.SetValue("")
			m.searching = false
			m.columnSel = 0
			m.search.Prompt = "> "
			m.columnSelMode = false
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// TODO: (willgorman) this should be only numbers up to the number of columns
			// and not sure what to do if more than 9 columns
			if m.columnSelMode && m.columnSel == 0 {
				col, _ := strconv.Atoi(msg.String()) // ignore error since we know it's a number
				m.columnSel = col
				m.searching = true
				log.Println(litter.Sdump("WTF", m.headers))
				m.search.Prompt = m.headers[col-1] + "> "
			}
		case "c":
			m.columnSelMode = true
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
	m = m.filterNodesBySearch().fillTable()
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
	// labelCols := map[string]int{}
	labelSet := map[string]struct{}{}
	// labelIdx := 2
	for _, n := range m.nodes {
		for l := range n.Labels {
			// _, ok := labelCols[l]
			// if ok {
			// 	continue
			// }
			// labelIdx++
			// labelCols[l] = labelIdx
			labelSet[l] = struct{}{}
		}
	}
	labels := maps.Keys(labelSet)
	slices.Sort(labels)

	m.headers = map[int]string{
		0: "Hostname",
		1: "IP",
		2: "OS",
	}
	for i, l := range labels {
		m.headers[i+3] = l
	}

	columns := make([]table.Column, len(labels)+3)
	columns[0] = table.Column{Title: m.title(m.headers[0], 1), Width: 30}
	columns[1] = table.Column{Title: m.title(m.headers[1], 2), Width: 16}
	columns[2] = table.Column{Title: m.title(m.headers[2], 3), Width: 30}
	for i, v := range labels {
		columns[i+3] = table.Column{Title: m.title(v, i+4), Width: 15}
	}

	// TODO: (willgorman) calculate widths by largest value in the column.  but what's the
	// ideal max width?
	m.table.SetColumns(columns)
	rows := []table.Row{}
	log.Println("VISIBLE: ", len(m.visible), " ALL: ", len(m.nodes))
	for _, n := range m.visible {
		row := make(table.Row, len(labels)+3)
		row[0] = n.Hostname
		row[1] = n.IP
		row[2] = n.OS
		for l, v := range n.Labels {
			idx := slices.Index(labels, l)
			if idx < 0 {
				continue
			}
			row[idx+3] = v
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
	log.Println("TBLE ROWS: ", len(rows))
	if len(rows) > 0 {
		log.Println("FRIST: ", rows[0][0])
	}
	log.Println("CURSES: ", m.table.Cursor())

	return m
}

func (m model) title(s string, i int) string {
	if m.columnSelMode {
		return strconv.Itoa(i)
	}
	return s
}

func (m model) filterNodesBySearch() model {
	if m.search.Value() == "" {
		m.visible = m.nodes
		return m
	}
	m.visible = nil

	if m.columnSel == 0 {
		txt2node := map[string]Node{}
		// if no column is selected we'll fuzzy search on all columns
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
		return m
	}

	txt2nodes := map[string][]Node{}
	for _, n := range m.nodes {
		switch m.columnSel {
		case 1:
			txt2nodes[strings.ToLower(n.Hostname)] = append(txt2nodes[strings.ToLower(n.Hostname)], n)
		case 2:
			txt2nodes[strings.ToLower(n.IP)] = append(txt2nodes[strings.ToLower(n.IP)], n)
		case 3:
			txt2nodes[strings.ToLower(n.OS)] = append(txt2nodes[strings.ToLower(n.OS)], n)
		default:
			txt2nodes[strings.ToLower(n.Labels[m.headers[m.columnSel-1]])] = append(txt2nodes[strings.ToLower(n.Labels[m.headers[m.columnSel-1]])], n)
		}
	}

	// FIXME: (willgorman) Need to make sure that m.visible ends up in a stable
	// order
	log.Println("SEARCHING: ", m.search.Value(), "IN: ", litter.Sdump(maps.Keys(txt2nodes)))
	ranks := fuzzy.RankFind(m.search.Value(), maps.Keys(txt2nodes))
	sort.Sort(ranks)
	log.Println("RESULTS: ", litter.Sdump(ranks))
	for _, rank := range ranks {
		nodes := txt2nodes[rank.Target]
		for _, n := range nodes {
			m.visible = append(m.visible, n)
		}

	}
	return m
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
	log.Println("------------------------------------")
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

	// make sure there's at least one profile in teleport,
	// if so then it will use that automatically, otherwise
	// user needs to login first
	if err := CheckProfiles(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
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
