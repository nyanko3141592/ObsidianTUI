package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/takahashinaoki/obsidiantui/config"
	"github.com/takahashinaoki/obsidiantui/internal/ui"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "obsidiantui [vault-path]",
		Short:   "A TUI client for Obsidian vaults",
		Long:    `ObsidianTUI is a terminal-based viewer and editor for Obsidian vaults.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	var vaultPath string
	if len(args) > 0 {
		vaultPath = args[0]
	} else {
		vaultPath = config.GetVaultPath()
	}

	if vaultPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		vaultPath = cwd
	}

	v, err := vault.NewVault(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to open vault at %s: %w", vaultPath, err)
	}

	config.SetVaultPath(vaultPath)
	if err := config.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	model := ui.NewModel(v)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
