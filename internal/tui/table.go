package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/willgorman/teash/internal/app"
)

const (
	colHostname = "hostname"
	colIP       = "ip"
	colOS       = "os"
)

func buildColumns(labelKeys []string, width int) []table.Column {
	// Calculate widths
	hostnameW := 30
	ipW := 16
	osW := 28

	cols := []table.Column{
		table.NewColumn(colHostname, "Hostname", hostnameW).WithFiltered(true),
		table.NewColumn(colIP, "IP", ipW).WithFiltered(true),
		table.NewColumn(colOS, "OS", osW).WithFiltered(true),
	}

	labelW := 16
	if len(labelKeys) > 0 && width > 0 {
		remaining := width - hostnameW - ipW - osW - 4 // borders/padding
		if remaining > 0 {
			perCol := remaining / len(labelKeys)
			if perCol > 8 {
				labelW = perCol
			}
		}
	}

	for _, key := range labelKeys {
		cols = append(cols, table.NewColumn(key, key, labelW).WithFiltered(true))
	}

	return cols
}

func buildRows(servers []app.ServerView, labelKeys []string) []table.Row {
	rows := make([]table.Row, len(servers))
	for i, sv := range servers {
		data := table.RowData{
			colHostname: sv.Hostname,
			colIP:       sv.Addr,
			colOS:       sv.OS,
		}
		for _, key := range labelKeys {
			data[key] = sv.AllLabels[key]
		}
		rows[i] = table.NewRow(data)
	}
	return rows
}

func buildTable(servers []app.ServerView, labelKeys []string, width, height int) table.Model {
	cols := buildColumns(labelKeys, width)
	rows := buildRows(servers, labelKeys)

	// The table chrome (top/bottom borders, header separator, footer
	// separator + pagination footer) always adds 6 lines on top of the
	// visible data rows, and the status bar below the table takes 1 more.
	// Under-reserving here causes the rendered output to exceed the
	// terminal height, which pushes the header off-screen via terminal
	// scrollback instead of the table's own (header-preserving) pagination.
	const tableChromeLines = 6
	const statusBarLines = 1
	tableHeight := max(height-tableChromeLines-statusBarLines, 3)

	t := table.New(cols).
		WithRows(rows).
		WithPageSize(tableHeight).
		Focused(true).
		SortByAsc(colHostname).
		WithBaseStyle(lipgloss.NewStyle().Align(lipgloss.Left))

	if width > 0 {
		t = t.WithTargetWidth(width)
	}

	return t
}
