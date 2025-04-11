package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Color definitions
var (

	// Additional styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(PrimaryColor).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(SecondaryColor).
			Padding(0, 1)
)

// AppState represents the different states of the application
type AppState int

const (
	StateInitial AppState = iota
	StateLoadingCredentials
	StateConfigGitHub
	StateSettingUpRemote
	StateSavingConfig
	StateSyncingFiles
	StateMonitoring
	StateExiting
)

// keyMap defines keybindings
type keyMap struct {
	Help  key.Binding
	Quit  key.Binding
	Enter key.Binding
	Back  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Quit},
		{k.Enter, k.Back},
	}
}

// defaultKeyMap returns a set of default keybindings
func defaultKeyMap() keyMap {
	return keyMap{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// AppModel represents the main application model
type AppModel struct {
	state        AppState
	viewport     viewport.Model
	spinner      spinner.Model
	keymap       keyMap
	help         help.Model
	width        int
	height       int
	logs         []string
	currentInput interface{}
	config       interface{}
	envConfig    interface{}
	gitRepo      interface{}
}

// NewApp creates a new application model
func NewApp() AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor)

	h := help.New()

	return AppModel{
		state:    StateInitial,
		viewport: vp,
		spinner:  s,
		keymap:   defaultKeyMap(),
		help:     h,
		logs:     []string{},
	}
}

// Init initializes the application model
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Sequence(
			tea.Printf("Starting application..."),
			tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
				return AppStateChangeMsg{NewState: StateLoadingCredentials}
			}),
		),
	)
}

// AppStateChangeMsg changes the application state
type AppStateChangeMsg struct {
	NewState AppState
}

// LogMsg adds a log message
type LogMsg struct {
	Text      string
	Type      string // "info", "success", "error", "warning"
	Timestamp time.Time
}

// AddLog adds a log message to the model
func (m *AppModel) AddLog(msg string, logType string) {
	prefix := ""
	switch logType {
	case "info":
		prefix = InfoPrefix
	case "success":
		prefix = SuccessPrefix
	case "error":
		prefix = ErrorPrefix
	case "warning":
		prefix = WarningPrefix
	}

	timestamp := time.Now().Format("15:04:05")
	formattedMsg := fmt.Sprintf("[%s] %s %s", timestamp, prefix, msg)
	m.logs = append(m.logs, formattedMsg)

	// Update viewport content
	m.viewport.SetContent(strings.Join(m.logs, "\n"))
	m.viewport.GotoBottom()
}

// Update handles UI updates
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 12 // Reserve space for header and footer
		m.help.Width = msg.Width

	case AppStateChangeMsg:
		m.state = msg.NewState
		switch m.state {
		case StateLoadingCredentials:
			m.AddLog("Loading credentials from .env file", "info")
			// In a real app, you'd create a command to actually load the credentials
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return LogMsg{Text: "Credentials loaded successfully", Type: "success"}
			})
		case StateConfigGitHub:
			m.AddLog("Configuring GitHub repository", "info")
		case StateSettingUpRemote:
			m.AddLog("Setting up Git remote", "info")
		case StateSavingConfig:
			m.AddLog("Saving configuration", "info")
		case StateSyncingFiles:
			m.AddLog("Synchronizing files", "info")
		case StateMonitoring:
			m.AddLog("Now monitoring for changes", "info")
		case StateExiting:
			m.AddLog("Exiting application", "info")
			return m, tea.Quit
		}

	case LogMsg:
		m.AddLog(msg.Text, msg.Type)

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Update spinner
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m AppModel) View() string {
	headerHeight := 5
	footerHeight := 3
	contentHeight := m.height - headerHeight - footerHeight

	// Header section
	header := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Linux Configuration Manager"),
		subtitleStyle.Render("Synchronize and manage your configuration files"),
		"",
	)

	// Status bar
	status := statusBarStyle.Width(m.width).Render(m.getStatusMessage())

	// Content/logs area
	var content string
	switch m.state {
	case StateInitial, StateLoadingCredentials, StateSettingUpRemote, StateSyncingFiles:
		spinner := m.spinner.View() + " "
		content = spinner + m.getStateMessage()
		content += "\n\n" + m.viewport.View()
	default:
		content = m.viewport.View()
	}

	content = lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width).
		Padding(0, 2).
		Render(content)

	// Footer with help
	help := helpStyle.Render(m.help.View(m.keymap))

	// Join all the sections
	ui := lipgloss.JoinVertical(lipgloss.Left,
		header,
		status,
		content,
		help,
	)

	return ui
}

// getStatusMessage returns a status message based on the current state
func (m AppModel) getStatusMessage() string {
	switch m.state {
	case StateInitial:
		return "Starting up..."
	case StateLoadingCredentials:
		return "Loading credentials..."
	case StateConfigGitHub:
		return "GitHub configuration"
	case StateSettingUpRemote:
		return "Setting up Git remote..."
	case StateSavingConfig:
		return "Saving configuration..."
	case StateSyncingFiles:
		return "Synchronizing files..."
	case StateMonitoring:
		return "Monitoring for changes (Press q to quit)"
	case StateExiting:
		return "Exiting..."
	default:
		return "Unknown state"
	}
}

// getStateMessage returns a more detailed message about the current state
func (m AppModel) getStateMessage() string {
	switch m.state {
	case StateInitial:
		return "Initializing application..."
	case StateLoadingCredentials:
		return "Loading credentials from .env file..."
	case StateConfigGitHub:
		return "Please configure your GitHub repository"
	case StateSettingUpRemote:
		return "Setting up Git remote repository..."
	case StateSavingConfig:
		return "Saving configuration to file..."
	case StateSyncingFiles:
		return "Synchronizing configuration files..."
	case StateMonitoring:
		return "Monitoring for configuration file changes"
	case StateExiting:
		return "Exiting application..."
	default:
		return "Unknown state"
	}
}

// RunApp starts the Bubble Tea application
func RunApp() {
	app := NewApp()
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running application:", err)
		os.Exit(1)
	}
}
