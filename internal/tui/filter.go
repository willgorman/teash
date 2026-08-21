package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/willgorman/teash/internal/app"
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

// columnSelector drives the two-level 'c' filter panel: a list of columns
// (level 1), and, once a column is chosen, a checklist of that column's
// distinct values (level 2, OR'd together when applied).
type columnSelector struct {
	columns  []string
	selected int // highlighted column in level 1

	inValues    bool // true = level 2 (value checklist) is active
	allValues   []string
	values      []string // allValues narrowed by input.Value()
	valueCursor int
	input       textinput.Model
}

func newColumnSelector(columns []string) columnSelector {
	ti := textinput.New()
	ti.Placeholder = "type to narrow..."
	ti.CharLimit = 64
	ti.Width = 40
	return columnSelector{
		columns: columns,
		input:   ti,
	}
}

// SelectedColumn returns the column highlighted in the level-1 list.
func (cs columnSelector) SelectedColumn() string {
	if cs.selected >= 0 && cs.selected < len(cs.columns) {
		return cs.columns[cs.selected]
	}
	return ""
}

// HighlightedValue returns the value highlighted in the level-2 checklist.
func (cs columnSelector) HighlightedValue() string {
	if cs.valueCursor >= 0 && cs.valueCursor < len(cs.values) {
		return cs.values[cs.valueCursor]
	}
	return ""
}

// enterValues switches to the level-2 checklist for the highlighted column.
func (cs columnSelector) enterValues(allValues []string) columnSelector {
	cs.inValues = true
	cs.allValues = allValues
	cs.values = allValues
	cs.valueCursor = 0
	cs.input.SetValue("")
	cs.input.Focus()
	return cs
}

// backToColumns returns from the level-2 checklist to the level-1 list.
func (cs columnSelector) backToColumns() columnSelector {
	cs.inValues = false
	cs.input.Blur()
	return cs
}

func (cs columnSelector) narrowValues() []string {
	query := strings.TrimSpace(cs.input.Value())
	if query == "" {
		return cs.allValues
	}
	matches := fuzzy.Find(query, cs.allValues)
	result := make([]string, len(matches))
	for i, match := range matches {
		result[i] = cs.allValues[match.Index]
	}
	return result
}

func (cs columnSelector) Update(msg tea.Msg) (columnSelector, tea.Cmd) {
	if !cs.inValues {
		return cs, nil
	}
	var cmd tea.Cmd
	cs.input, cmd = cs.input.Update(msg)
	cs.values = cs.narrowValues()
	if cs.valueCursor >= len(cs.values) {
		cs.valueCursor = max(len(cs.values)-1, 0)
	}
	return cs, cmd
}

func valuesFor(filters []app.ColumnFilter, col string) []string {
	for _, f := range filters {
		if f.Column == col {
			return f.Values
		}
	}
	return nil
}

func (cs columnSelector) View(filters []app.ColumnFilter) string {
	if cs.inValues {
		return cs.viewValues(valuesFor(filters, cs.SelectedColumn()))
	}
	return cs.viewColumns(filters)
}

func (cs columnSelector) viewColumns(filters []app.ColumnFilter) string {
	var sb strings.Builder
	sb.WriteString(stylePrompt.Render("Select column (enter: choose values, x: clear, esc: close):") + "\n")
	for i, col := range cs.columns {
		marker := "  "
		if i == cs.selected {
			marker = "> "
		}
		summary := ""
		if vals := valuesFor(filters, col); len(vals) > 0 {
			summary = "  " + styleHelp.Render(strings.Join(vals, ", "))
		}
		fmt.Fprintf(&sb, "%s%s%s\n", marker, col, summary)
	}
	return sb.String()
}

func (cs columnSelector) viewValues(selectedValues []string) string {
	selectedSet := make(map[string]bool, len(selectedValues))
	for _, v := range selectedValues {
		selectedSet[v] = true
	}
	var sb strings.Builder
	sb.WriteString(stylePrompt.Render(fmt.Sprintf("Filter %s (space: toggle, enter/esc: back):", cs.SelectedColumn())) + "\n")
	sb.WriteString(cs.input.View() + "\n")
	for i, v := range cs.values {
		box := "[ ]"
		if selectedSet[v] {
			box = "[x]"
		}
		marker := "  "
		if i == cs.valueCursor {
			marker = "> "
		}
		fmt.Fprintf(&sb, "%s%s %s\n", marker, box, v)
	}
	if len(cs.values) == 0 {
		sb.WriteString(styleHelp.Render("  (no matches)") + "\n")
	}
	return sb.String()
}
