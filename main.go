// Copyright (c) 2021, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"fmt"
	"os"

	"github.com/bhmj/jsonslice"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

const (
	initialInputs = 2
	maxInputs     = 2
	minInputs     = 1
	helpHeight    = 5
)

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	cursorLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230"))

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

	endOfBufferStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("235"))

	focusedPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false)
)

type keymap = struct {
	next, delete, quit key.Binding
}

func newTextarea() textarea.Model {
	t := textarea.New()
	t.Prompt = ""
	t.Placeholder = "Type something"
	t.ShowLineNumbers = true
	t.Cursor.Style = cursorStyle
	t.FocusedStyle.Placeholder = focusedPlaceholderStyle
	t.BlurredStyle.Placeholder = placeholderStyle
	t.FocusedStyle.CursorLine = cursorLineStyle
	t.FocusedStyle.Base = focusedBorderStyle
	t.BlurredStyle.Base = blurredBorderStyle
	t.FocusedStyle.EndOfBuffer = endOfBufferStyle
	t.BlurredStyle.EndOfBuffer = endOfBufferStyle
	t.KeyMap.DeleteWordBackward.SetEnabled(false)
	t.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
	t.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))
	t.Blur()
	return t
}

func newTextareaResults() textarea.Model {
	t := textarea.New()
	t.Prompt = ""
	t.Placeholder = "Results"
	t.ShowLineNumbers = false
	t.Cursor.Style = cursorStyle
	t.FocusedStyle.Placeholder = focusedPlaceholderStyle
	t.BlurredStyle.Placeholder = placeholderStyle
	t.FocusedStyle.CursorLine = cursorLineStyle
	t.FocusedStyle.Base = focusedBorderStyle
	t.FocusedStyle.EndOfBuffer = endOfBufferStyle
	t.BlurredStyle.EndOfBuffer = endOfBufferStyle
	t.KeyMap.DeleteWordBackward.SetEnabled(false)
	t.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
	t.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))
	t.Blur()
	return t
}

type editorModel struct {
	textInput    textinput.Model
	width        int
	height       int
	keymap       keymap
	help         help.Model
	initialInput []byte
	input        textarea.Model
	result       textarea.Model
}

func (m editorModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.textInput.Blur()
			m.input.Blur()
			m.result.Blur()
			return m, tea.Quit
		case key.Matches(msg, m.keymap.next):
			if m.input.Focused() {
				m.input.Blur()
				m.textInput.Focus()
				m.keymap.next.SetHelp("tab", "(switch to jsonpath query)")
			} else {
				m.input.Focus()
				m.textInput.Blur()
				m.keymap.next.SetHelp("tab", "(switch to editor)")
			}
		case key.Matches(msg, m.keymap.delete):
			m.input.SetValue("")
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	m.sizeInputs()

	var cmd tea.Cmd

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// Call jsonslice and evaluate path

	result, err := jsonslice.Get(m.initialInput, m.textInput.Value())

	if err != nil {
		m.result.SetValue(err.Error())
	} else {
		m.result.SetValue(string(result))
	}
	return m, tea.Batch(cmds...)
}

func (m *editorModel) sizeInputs() {
	m.input.SetWidth(m.width / 2)
	m.input.SetHeight(m.height - helpHeight - 6)

	m.result.SetWidth(m.width / 2)
	m.result.SetHeight(m.height - helpHeight - 6)
}

func (m editorModel) View() string {
	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.next,
		m.keymap.delete,
		m.keymap.quit,
	})

	title := lipgloss.NewStyle().Width(40).MaxWidth(40).Height(2).MarginTop(1).Render("Enter jsonPath Query")

	input := lipgloss.NewStyle().Width(40).Height(2).Render(m.textInput.View())

	leftAreaTitle := lipgloss.NewStyle().Bold(true).Height(1).Render("Input")

	rightAreaTitle := lipgloss.NewStyle().Bold(true).Height(1).Render("Results")

	left := lipgloss.NewStyle().Height(30).Render(m.input.View())
	right := lipgloss.NewStyle().Height(30).Render(m.result.View())

	leftSection := lipgloss.JoinVertical(lipgloss.Left, leftAreaTitle, left)
	rightSection := lipgloss.JoinVertical(lipgloss.Left, rightAreaTitle, right)

	splitStyle := lipgloss.NewStyle().Width(m.width / 2)
	leftStyled := splitStyle.Render(leftSection)
	rightStyled := splitStyle.Render(rightSection)

	textAreas := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	return lipgloss.JoinVertical(lipgloss.Left, title, input, textAreas, help)
}

func newModel(defaultJson []byte) editorModel {
	ti := textinput.New()
	ti.Placeholder = "$.requestedItemsStatus[0].id"
	ti.SetValue("$.requestedItemsStatus[0].id")
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 500

	m := editorModel{
		textInput: ti,
		input:     newTextarea(),
		result:    newTextareaResults(),
		help:      help.New(),
		keymap: keymap{
			next: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "(switch to editor)"),
			),
			delete: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "Clear editor"),
			),
			quit: key.NewBinding(
				key.WithKeys("esc", "ctrl+c"),
				key.WithHelp("esc", "quit"),
			),
		},
	}

	if defaultJson != nil {
		m.initialInput = defaultJson
		m.input.SetValue(string(defaultJson))
		m.textInput.SetValue("$")
	} else {
		m.input.SetValue("{\r    \"accessRequestId\": \"2c91808b6ef1d43e016efba0ce470904\",\r    \"requestedFor\": {\r        \"type\": \"IDENTITY\",\r        \"id\": \"2c91808568c529c60168cca6f90c1313\",\r        \"name\": \"William Wilson\"\r    },\r    \"requestedItemsStatus\": [\r        {\r            \"id\": \"2c91808b6ef1d43e016efba0ce470904\",\r            \"name\": \"Engineering Access\",\r            \"description\": \"Access to engineering database\",\r            \"type\": \"ACCESS_PROFILE\",\r            \"operation\": \"Add\",\r            \"comment\": \"William needs this access to do his job.\",\r            \"clientMetadata\": {\r                \"applicationName\": \"My application\"\r            },\r            \"approvalInfo\": [\r                {\r                    \"approvalComment\": \"This access looks good.  Approved.\",\r                    \"approvalDecision\": \"APPROVED\",\r                    \"approverName\": \"Stephen.Austin\",\r                    \"approver\": {\r                        \"type\": \"IDENTITY\",\r                        \"id\": \"2c91808568c529c60168cca6f90c1313\",\r                        \"name\": \"William Wilson\"\r                    }\r                }\r            ]\r        }\r    ],\r    \"requestedBy\": {\r        \"type\": \"IDENTITY\",\r        \"id\": \"2c91808568c529c60168cca6f90c1313\",\r        \"name\": \"William Wilson\"\r    }\r}")
	}

	m.textInput.Focus()
	return m
}

func main() {

	var data []byte
	var err error
	filepath := "testdata.json"

	data, err = os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := tea.NewProgram(newModel(data), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error while running program:", err)
		log.Fatal(err)
	}
}
