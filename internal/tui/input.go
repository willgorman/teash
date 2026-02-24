package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// labelInput handles the text input for adding a label (key:value format).
type labelInput struct {
	input    textinput.Model
	hostname string
}

func newLabelInput(hostname string) labelInput {
	ti := textinput.New()
	ti.Placeholder = "key:value"
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = stylePrompt.Render(fmt.Sprintf("Label for %s > ", hostname))
	return labelInput{
		input:    ti,
		hostname: hostname,
	}
}

type labelAddedMsg struct {
	hostname string
	key      string
	value    string
}

func (li labelInput) Update(msg tea.Msg) (labelInput, tea.Cmd) {
	var cmd tea.Cmd
	li.input, cmd = li.input.Update(msg)
	return li, cmd
}

func (li labelInput) View() string {
	return li.input.View()
}

func (li labelInput) Value() (key, value string, err error) {
	text := strings.TrimSpace(li.input.Value())
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid format: use key:value")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
