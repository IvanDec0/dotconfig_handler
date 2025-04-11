package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styling definitions
var (
	// Log prefix symbols
	InfoPrefix    = "ℹ"
	SuccessPrefix = "✓"
	ErrorPrefix   = "✗"
	WarningPrefix = "⚠"

	// Base colors
	PrimaryColor   = lipgloss.Color("#89B4FA") // Cyan
	SecondaryColor = lipgloss.Color("#F9E2AF") // Yellow
	SuccessColor   = lipgloss.Color("#A6E3A1") // Green
	ErrorColor     = lipgloss.Color("#F38BA8") // Red
	WarningColor   = lipgloss.Color("#F9E2AF") // Yellow
	InfoColor      = lipgloss.Color("#89B4FA") // Blue
	PromptColor    = lipgloss.Color("#CBA6F7") // Magenta
	HighlightColor = lipgloss.Color("#F5F5F5") // White

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	promptStyle = lipgloss.NewStyle().
			Foreground(PromptColor)

	highlightStyle = lipgloss.NewStyle().
			Foreground(HighlightColor).
			Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor)

	// Operation styles
	addedStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	modifiedStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	deletedStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 1)
)

// ASCII art logo for the application
const logo = `
   ____             __ _          _   _                 _ _           
  / ___|___  _ __  / _(_) __ _   | | | | __ _ _ __   __| | | ___ _ __ 
 | |   / _ \| '_ \| |_| |/ _' |  | |_| |/ _' | '_ \ / _' | |/ _ \ '__|
 | |__| (_) | | | |  _| | (_| |  |  _  | (_| | | | | (_| | |  __/ |   
  \____\___/|_| |_|_| |_|\__, |  |_| |_|\__,_|_| |_|\__,_|_|\___|_|   
                         |___/                                        
`

// ---------- Bubble Tea Models ----------

// YesNoPromptModel represents a yes/no prompt
type YesNoPromptModel struct {
	question     string
	defaultValue bool
	response     bool
	quitting     bool
}

// Init initializes the model
func (m YesNoPromptModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m YesNoPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.response = true
			m.quitting = true
			return m, tea.Quit
		case "n", "N":
			m.response = false
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.response = m.defaultValue
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c", "q":
			m.response = m.defaultValue
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m YesNoPromptModel) View() string {
	defaultText := "Y/n"
	if !m.defaultValue {
		defaultText = "y/N"
	}

	var view strings.Builder

	if m.quitting {
		view.WriteString(promptStyle.Render("❯ ") + m.question + " [" + defaultText + "]: ")
		if m.response {
			view.WriteString("Yes")
		} else {
			view.WriteString("No")
		}
		view.WriteString("\n")
	} else {
		view.WriteString(promptStyle.Render("❯ ") + m.question + " [" + defaultText + "]: ")
	}

	return view.String()
}

// TextInputModel represents a text input prompt
type TextInputModel struct {
	textInput    textinput.Model
	question     string
	defaultValue string
	response     string
	quitting     bool
}

// NewTextInputModel creates a new text input model
func NewTextInputModel(question, defaultValue string) TextInputModel {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return TextInputModel{
		textInput:    ti,
		question:     question,
		defaultValue: defaultValue,
	}
}

// Init initializes the model
func (m TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles user input
func (m TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			value := m.textInput.Value()
			if value == "" && m.defaultValue != "" {
				m.response = m.defaultValue
			} else {
				m.response = value
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.response = m.defaultValue
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the model
func (m TextInputModel) View() string {
	if m.quitting {
		return promptStyle.Render("❯ ") + m.question + ": " + m.response + "\n"
	}

	var prompt string
	if m.defaultValue != "" {
		prompt = promptStyle.Render("❯ ") + m.question + " [" + m.defaultValue + "]: "
	} else {
		prompt = promptStyle.Render("❯ ") + m.question + ": "
	}

	return prompt + m.textInput.View()
}

// ProgressModel represents a loading progress indicator
type ProgressModel struct {
	spinner  spinner.Model
	message  string
	duration time.Duration
	start    time.Time
	done     bool
}

// NewProgressModel creates a new progress model
func NewProgressModel(message string, duration time.Duration) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(InfoColor)

	return ProgressModel{
		spinner:  s,
		message:  message,
		duration: duration,
	}
}

// Init initializes the model
func (m ProgressModel) Init() tea.Cmd {
	m.start = time.Now()
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return tickMsg{}
		}),
	)
}

type tickMsg struct{}

// Update handles updates
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		if time.Since(m.start) >= m.duration {
			m.done = true
			return m, tea.Quit
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return tickMsg{}
		})
	case tea.KeyMsg:
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

// View renders the model
func (m ProgressModel) View() string {
	if m.done {
		return successStyle.Render("✓ ") + m.message + "\n"
	}
	return m.spinner.View() + " " + m.message
}

// SelectModel represents a selection prompt
type SelectModel struct {
	choices  []string
	cursor   int
	selected int
	question string
	quitting bool
}

// Init initializes the model
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m SelectModel) View() string {
	var s strings.Builder

	if m.quitting && m.selected >= 0 && m.selected < len(m.choices) {
		s.WriteString(promptStyle.Render("❯ ") + m.question + ": " + m.choices[m.selected] + "\n")
		return s.String()
	}

	s.WriteString(promptStyle.Render("❯ ") + m.question + ":\n\n")

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			choice = highlightStyle.Render(choice)
		} else {
			choice = infoStyle.Render(choice)
		}
		s.WriteString(fmt.Sprintf("  %s %s\n", cursor, choice))
	}

	s.WriteString("\n(Use arrow keys to navigate, Enter to select)")
	return s.String()
}

// ---------- Public UI Functions ----------

// PrintLogo prints the application logo
func PrintLogo() {
	fmt.Println(titleStyle.Render(logo))
}

// PrintTitle prints a title with a decorative border
func PrintTitle(title string) {
	fmt.Println()
	boxedTitle := boxStyle.Render(highlightStyle.Render(title))
	fmt.Println(boxedTitle)
	fmt.Println()
}

// PrintSection prints a section title
func PrintSection(title string) {
	fmt.Println()
	fmt.Println(subtitleStyle.Render("▶ " + title))
	fmt.Println(separatorStyle.Render(strings.Repeat("─", len(title)+3)))
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(successStyle.Render("✓ " + message))
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Println(errorStyle.Render("✗ " + message))
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println(warningStyle.Render("⚠ " + message))
}

// PrintInfo prints an informational message
func PrintInfo(message string) {
	fmt.Println(infoStyle.Render("ℹ " + message))
}

// PrintPrompt prints a user prompt
func PrintPrompt(prompt string) {
	fmt.Print(promptStyle.Render("❯ " + prompt + " "))
}

// PrintProgress shows a loading animation for a specified duration
func PrintProgress(message string, duration time.Duration) {
	p := NewProgressModel(message, duration)
	if _, err := tea.NewProgram(p).Run(); err != nil {
		// Fall back to simple output if the TUI fails
		fmt.Printf("Loading: %s...\n", message)
		time.Sleep(duration)
		fmt.Printf("Done: %s\n", message)
	}
}

// PrintFileOperation prints information about file operations
func PrintFileOperation(operation, path string) {
	switch operation {
	case "added":
		fmt.Println(addedStyle.Render("  [+] " + path))
	case "modified":
		fmt.Println(modifiedStyle.Render("  [~] " + path))
	case "deleted":
		fmt.Println(deletedStyle.Render("  [-] " + path))
	default:
		fmt.Println("  [?] " + path)
	}
}

// PrintSeparator prints a separator line
func PrintSeparator() {
	fmt.Println(separatorStyle.Render(strings.Repeat("─", 50)))
}

// FormatCommitMessage formats a commit message with colors
func FormatCommitMessage(message string) string {
	// Highlight the added/modified/deleted keywords
	message = strings.ReplaceAll(message, "added:", addedStyle.Render("added:"))
	message = strings.ReplaceAll(message, "modified:", modifiedStyle.Render("modified:"))
	message = strings.ReplaceAll(message, "deleted:", deletedStyle.Render("deleted:"))

	return message
}

// PromptYesNo prompts the user for a yes/no response with a default value
func PromptYesNo(prompt string, defaultValue bool) bool {
	p := YesNoPromptModel{
		question:     prompt,
		defaultValue: defaultValue,
	}

	m, err := tea.NewProgram(p).Run()
	if err != nil {
		// Fall back to console-based prompt if TUI fails
		fmt.Print(promptStyle.Render("❯ " + prompt + " [" + (map[bool]string{true: "Y/n", false: "y/N"})[defaultValue] + "]: "))
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			return defaultValue
		}
		return input == "y" || input == "yes"
	}

	model, _ := m.(YesNoPromptModel)
	return model.response
}

// PromptInput prompts the user for text input with an optional default value
func PromptInput(prompt string, defaultValue string) string {
	model := NewTextInputModel(prompt, defaultValue)

	m, err := tea.NewProgram(model).Run()
	if err != nil {
		// Fall back to console-based prompt if TUI fails
		defaultText := ""
		if defaultValue != "" {
			defaultText = " [" + defaultValue + "]"
		}
		fmt.Print(promptStyle.Render("❯ " + prompt + defaultText + ": "))
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" && defaultValue != "" {
			return defaultValue
		}
		return input
	}

	finalModel, _ := m.(TextInputModel)
	return finalModel.response
}

// PromptSelect prompts the user to select an option from a list of choices
func PromptSelect(prompt string, choices []string, defaultIndex int) string {
	if defaultIndex < 0 || defaultIndex >= len(choices) {
		defaultIndex = 0
	}

	p := SelectModel{
		question: prompt,
		choices:  choices,
		cursor:   defaultIndex,
		selected: defaultIndex,
	}

	m, err := tea.NewProgram(p).Run()
	if err != nil {
		// Fall back to simple prompt if TUI fails
		fmt.Println(promptStyle.Render("❯ " + prompt + ":"))
		for i, choice := range choices {
			fmt.Printf("  %d. %s\n", i+1, choice)
		}
		fmt.Print("Enter your choice (1-" + fmt.Sprintf("%d", len(choices)) + "): ")
		var input int
		fmt.Scanln(&input)
		if input > 0 && input <= len(choices) {
			return choices[input-1]
		}
		return choices[defaultIndex]
	}

	model, _ := m.(SelectModel)
	if model.selected >= 0 && model.selected < len(choices) {
		return choices[model.selected]
	}
	return choices[defaultIndex]
}

// ExecuteCommand runs a shell command and returns its output
func ExecuteCommand(command string, args ...string) (string, error) {
	PrintInfo(fmt.Sprintf("Executing: %s %s", command, strings.Join(args, " ")))

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		PrintError(fmt.Sprintf("Command failed: %v", err))
		return string(output), err
	}

	return string(output), nil
}
