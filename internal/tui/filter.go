package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// searchInput handles the global search text input.
type searchInput struct {
	input textinput.Model
}

func newSearchInput() searchInput {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40
	ti.Prompt = stylePrompt.Render("/ ")
	return searchInput{input: ti}
}

func (si searchInput) Update(msg tea.Msg) (searchInput, tea.Cmd) {
	var cmd tea.Cmd
	si.input, cmd = si.input.Update(msg)
	return si, cmd
}

func (si searchInput) View() string {
	return si.input.View()
}

func (si searchInput) Value() string {
	return strings.TrimSpace(si.input.Value())
}

// columnSelector handles the column selection and value filter for 'c' key.
type columnSelector struct {
	columns  []string
	selected int
	input    textinput.Model
	choosing bool // true = choosing column, false = entering value
}

func newColumnSelector(columns []string) columnSelector {
	ti := textinput.New()
	ti.Placeholder = "filter value..."
	ti.CharLimit = 64
	ti.Width = 40
	return columnSelector{
		columns:  columns,
		selected: 0,
		input:    ti,
		choosing: true,
	}
}

func (cs columnSelector) Update(msg tea.Msg) (columnSelector, tea.Cmd) {
	if !cs.choosing {
		var cmd tea.Cmd
		cs.input, cmd = cs.input.Update(msg)
		return cs, cmd
	}
	return cs, nil
}

func (cs columnSelector) View() string {
	if cs.choosing {
		var sb strings.Builder
		sb.WriteString(stylePrompt.Render("Select column to filter:") + "\n")
		for i, col := range cs.columns {
			if i == cs.selected {
				sb.WriteString(fmt.Sprintf("  > %s\n", col))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", col))
			}
		}
		return sb.String()
	}
	cs.input.Prompt = stylePrompt.Render(fmt.Sprintf("Filter %s > ", cs.columns[cs.selected]))
	return cs.input.View()
}

func (cs columnSelector) SelectedColumn() string {
	if cs.selected >= 0 && cs.selected < len(cs.columns) {
		return cs.columns[cs.selected]
	}
	return ""
}

func (cs columnSelector) FilterValue() string {
	return strings.TrimSpace(cs.input.Value())
}
