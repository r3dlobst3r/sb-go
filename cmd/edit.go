package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/r3dlobst3r/sb-go/internal/constants"
	"github.com/r3dlobst3r/sb-go/internal/signals"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

// ConfigItem represents a configuration file option
type ConfigItem struct {
	title       string
	description string
	path        string
}

// Title returns the title for the list item
func (i ConfigItem) Title() string { return i.title }

// Description returns the description for the list item
func (i ConfigItem) Description() string { return i.description }

// FilterValue returns the filter value for the list item
func (i ConfigItem) FilterValue() string { return i.title }

// Styling
var docStyle = lipgloss.NewStyle().Margin(1, 2)

// ConfigSelectorModel represents the bubbletea model for selecting configuration files
type ConfigSelectorModel struct {
	list list.Model
}

func (m ConfigSelectorModel) Init() tea.Cmd {
	return nil
}

// editorFinishedMsg is sent when the editor process exits.
type editorFinishedMsg struct{ err error }

func (m ConfigSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			signals.GetGlobalManager().Shutdown(130)
			return m, tea.Quit
		}
		if msg.String() == "q" {
			return m, tea.Quit
		}

		if msg.String() == "enter" {
			selectedItem, ok := m.list.SelectedItem().(ConfigItem)
			if ok {
				c, err := editorCommand(selectedItem.path)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return m, tea.Quit
				}
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return editorFinishedMsg{err}
				})
			}
		}
	case editorFinishedMsg:
		if msg.err != nil {
			fmt.Printf("Error: %v\n", msg.err)
		}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ConfigSelectorModel) View() tea.View {
	v := tea.NewView("\nSelect a Saltbox configuration file to edit:\n\n" + m.list.View())
	v.AltScreen = true
	return v
}

// editorCommand builds an *exec.Cmd for the user's preferred editor.
func editorCommand(path string) (*exec.Cmd, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not yet exist: %s", path)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default to nano if EDITOR is not set
	}

	// Validate and sanitize editor command
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return nil, fmt.Errorf("invalid EDITOR environment variable")
	}

	// Get the absolute path of the editor executable
	editorPath, err := exec.LookPath(editorParts[0])
	if err != nil {
		if filepath.IsAbs(editorParts[0]) {
			editorPath = editorParts[0]
		} else {
			return nil, fmt.Errorf("editor '%s' not found in PATH", editorParts[0])
		}
	}

	// Construct command with validated editor and additional args if any
	var args []string
	if len(editorParts) > 1 {
		args = append(editorParts[1:], path)
	} else {
		args = []string{path}
	}

	return exec.Command(editorPath, args...), nil
}

func openEditor(path string) error {
	c, err := editorCommand(path)
	if err != nil {
		return err
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("error opening editor: %w", err)
	}
	return nil
}

func runBubbleTeaList() error {
	configItems := []list.Item{
		ConfigItem{
			title:       "Accounts",
			description: "accounts.yml",
			path:        constants.SaltboxAccountsConfigPath,
		},
		ConfigItem{
			title:       "Settings",
			description: "settings.yml",
			path:        constants.SaltboxSettingsConfigPath,
		},
		ConfigItem{
			title:       "Advanced Settings",
			description: "adv_settings.yml",
			path:        constants.SaltboxAdvancedSettingsConfigPath,
		},
		ConfigItem{
			title:       "Backup Settings",
			description: "backup_config.yml",
			path:        constants.SaltboxBackupConfigPath,
		},
		ConfigItem{
			title:       "Hetzner VLAN Settings",
			description: "hetzner_vlan.yml",
			path:        constants.SaltboxHetznerVLANConfigPath,
		},
		ConfigItem{
			title:       "Inventory Settings",
			description: "localhost.yml",
			path:        constants.SaltboxInventoryConfigPath,
		},
	}

	// Initialize a list with proper dimensions
	delegate := list.NewDefaultDelegate()
	m := ConfigSelectorModel{list: list.New(configItems, delegate, 30, 10)} // Set width and height
	m.list.SetShowTitle(false)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)
	m.list.SetShowHelp(true)
	m.list.SetShowFilter(false)
	m.list.SetShowPagination(false)

	// Get terminal dimensions
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running bubbletea program: %w", err)
	}
	return nil
}

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit Saltbox configuration files",
	Long:  `Edit Saltbox configuration files using your default editor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if err := runBubbleTeaList(); err != nil {
				return err
			}
			return nil
		}

		switch args[0] {
		case "accounts":
			return openEditor(constants.SaltboxAccountsConfigPath)
		case "adv_settings":
			return openEditor(constants.SaltboxAdvancedSettingsConfigPath)
		case "backup_config":
			return openEditor(constants.SaltboxBackupConfigPath)
		case "hetzner_vlan":
			return openEditor(constants.SaltboxHetznerVLANConfigPath)
		case "inventory":
			return openEditor(constants.SaltboxInventoryConfigPath)
		case "settings":
			return openEditor(constants.SaltboxSettingsConfigPath)
		default:
			// Use lipgloss to prevent customErrorHandler from transforming the message
			normalStyle := lipgloss.NewStyle()
			msg := fmt.Sprintf("%s\n%s",
				normalStyle.Render(fmt.Sprintf("Unknown configuration: %s", args[0])),
				normalStyle.Render("Run 'sb edit' to see all available configurations"))
			return fmt.Errorf("%s", msg)
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	// Subcommands for each configuration file
	editCmd.AddCommand(&cobra.Command{
		Use:   "accounts",
		Short: "Accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxAccountsConfigPath)
		},
	})

	editCmd.AddCommand(&cobra.Command{
		Use:   "adv_settings",
		Short: "Advanced Settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxAdvancedSettingsConfigPath)
		},
	})

	editCmd.AddCommand(&cobra.Command{
		Use:   "backup_config",
		Short: "Backup Settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxBackupConfigPath)
		},
	})

	editCmd.AddCommand(&cobra.Command{
		Use:   "hetzner_vlan",
		Short: "Hetzner VLAN Settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxHetznerVLANConfigPath)
		},
	})

	editCmd.AddCommand(&cobra.Command{
		Use:   "settings",
		Short: "Settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxSettingsConfigPath)
		},
	})

	editCmd.AddCommand(&cobra.Command{
		Use:   "inventory",
		Short: "Inventory Settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor(constants.SaltboxInventoryConfigPath)
		},
	})
}
