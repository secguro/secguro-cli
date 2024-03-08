package main

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type modelChooseOption struct {
	prompt  string
	choices []string
	cursor  int
	choice  string
}

func (m modelChooseOption) Init() tea.Cmd {
	return nil
}

func (m modelChooseOption) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint: ireturn // must be like this
	switch msg := msg.(type) { //nolint: gocritic
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = m.choices[m.cursor]
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.choices) - 1
			}
		}
	}

	return m, nil
}

func (m modelChooseOption) View() string {
	s := strings.Builder{}
	s.WriteString(m.prompt + "\n\n")

	for i := 0; i < len(m.choices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(m.choices[i])
		s.WriteString("\n")
	}
	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

func initialModelChooseOption(prompt string, choices []string) modelChooseOption {
	return modelChooseOption{
		prompt:  prompt,
		choices: choices,
		cursor:  0,
		choice:  choices[0],
	}
}

func getOptionChoice(prompt string, choices []string) (int, string, error) {
	if len(choices) == 0 {
		return 0, "", errors.New("empty array given for choices")
	}

	p := tea.NewProgram(initialModelChooseOption(prompt, choices), tea.WithAltScreen())

	// Run returns the model as a tea.Model.
	m, err := p.Run()
	if err != nil {
		return 0, "", err
	}

	// Assert the final tea.Model to our local model and print the choice.
	if m, ok := m.(modelChooseOption); ok && m.choice != "" {
		return m.cursor, m.choice, nil
	}

	return 0, "", errors.New("option chooser terminated unexpectedly")
}