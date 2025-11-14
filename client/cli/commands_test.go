package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	t.Run("should have correct basic properties", func(t *testing.T) {
		assert.Equal(t, "gophkeeper", RootCmd.Use)
		assert.Equal(t, "Secure password manager", RootCmd.Short)
	})

	t.Run("should not have subcommands before Execute", func(t *testing.T) {
		// Before Execute is called, RootCmd should not have subcommands
		commands := RootCmd.Commands()
		assert.Empty(t, commands, "RootCmd should not have subcommands before Execute")
	})
}

func TestExecute(t *testing.T) {
	t.Run("should set version and build date", func(t *testing.T) {
		version := "v1.0.0"
		buildDate := "2024-01-01"

		// Reset global variables
		Version = ""
		BuildDate = ""

		// Execute with a simple command structure
		err := Execute(version, buildDate)

		assert.NoError(t, err)
		assert.Equal(t, version, Version)
		assert.Equal(t, buildDate, BuildDate)
	})

	t.Run("should add all commands during execution", func(t *testing.T) {
		// Create a fresh root command for testing
		testRoot := &cobra.Command{
			Use:   "gophkeeper",
			Short: "Secure password manager",
		}

		// Temporarily replace RootCmd
		originalRoot := RootCmd
		RootCmd = testRoot
		defer func() { RootCmd = originalRoot }()

		// Execute to add commands
		err := Execute("v1.0.0", "2024-01-01")
		assert.NoError(t, err)

		// Check that commands were added
		commands := testRoot.Commands()
		assert.NotEmpty(t, commands, "Commands should be added during Execute")

		// Check for some key commands
		commandNames := make([]string, len(commands))
		for i, cmd := range commands {
			commandNames[i] = cmd.Name()
		}

		// Check for essential commands
		essentialCommands := []string{"version", "register", "login", "logout"}
		for _, cmd := range essentialCommands {
			assert.Contains(t, commandNames, cmd)
		}
	})
}

func TestCommandInitialization(t *testing.T) {
	t.Run("should initialize version command correctly", func(t *testing.T) {
		assert.Equal(t, "version", versionCmd.Use)
		assert.Equal(t, "Show version and configuration information", versionCmd.Short)
		assert.NotNil(t, versionCmd.Run)
	})

	t.Run("should initialize auth commands correctly", func(t *testing.T) {
		assert.Equal(t, "register", registerCmd.Use)
		assert.Equal(t, "Register a new user", registerCmd.Short)

		assert.Equal(t, "login", loginCmd.Use)
		assert.Equal(t, "Login to GophKeeper", loginCmd.Short)

		assert.Equal(t, "logout", logoutCmd.Use)
		assert.Equal(t, "Logout and clear saved token", logoutCmd.Short)

		assert.Equal(t, "status", statusCmd.Use)
		assert.Equal(t, "Show authentication status", statusCmd.Short)
	})

	t.Run("should have command flags defined", func(t *testing.T) {
		// Test that register command has login flag
		registerFlag := registerCmd.Flags().Lookup("login")
		require.NotNil(t, registerFlag)
		assert.Equal(t, "l", registerFlag.Shorthand)

		// Test that login command has login flag
		loginFlag := loginCmd.Flags().Lookup("login")
		require.NotNil(t, loginFlag)
		assert.Equal(t, "l", loginFlag.Shorthand)
	})
}
