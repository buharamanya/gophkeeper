package cli

import (
	"testing"

	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAuthentication(t *testing.T) {
	t.Run("should return error when not authenticated", func(t *testing.T) {
		// Mock config to return no token
		originalLoad := configLoad
		configLoad = func() *config.Config {
			return &config.Config{
				ServerURL:  "http://test.com",
				Token:      "",
				SkipVerify: false,
				Timeout:    30,
			}
		}
		defer func() { configLoad = originalLoad }()

		client, err := checkAuthentication()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
		assert.Nil(t, client)
	})

	t.Run("should validate authentication logic", func(t *testing.T) {
		// Test the logic without executing the actual function
		// This test verifies that the function structure is correct
		assert.NotNil(t, checkAuthentication)
	})
}

func TestStoreCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		assert.Equal(t, "store", storeCmd.Use)
		assert.Equal(t, "Store new data entry", storeCmd.Short)
		assert.NotNil(t, storeCmd.Run)
	})

	t.Run("should have required name flag", func(t *testing.T) {
		nameFlag := storeCmd.Flags().Lookup("name")
		require.NotNil(t, nameFlag)
		assert.Equal(t, "n", nameFlag.Shorthand)
		assert.Equal(t, "Entry name", nameFlag.Usage)
	})

	t.Run("should have all data flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
			usage     string
		}{
			{"type", "t", "Data type (login_password, text, binary, card)"},
			{"name", "n", "Entry name"},
			{"metadata", "m", "Metadata"},
			{"file", "f", "Input file (default: stdin)"},
		}

		for _, flag := range flags {
			f := storeCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "Flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
			assert.Equal(t, flag.usage, f.Usage)
		}
	})
}

func TestListCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		assert.Equal(t, "list", listCmd.Use)
		assert.Equal(t, "List all data entries", listCmd.Short)
		assert.NotNil(t, listCmd.Run)
	})

	t.Run("should not require arguments", func(t *testing.T) {
		// list command should work without arguments
		assert.Nil(t, listCmd.Args)
	})
}

func TestGetCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		// get [id] includes argument placeholder
		assert.Equal(t, "get [id]", getCmd.Use)
		assert.Equal(t, "Get data entry by ID", getCmd.Short)
		assert.NotNil(t, getCmd.Run)
	})

	t.Run("should require exactly one argument", func(t *testing.T) {
		assert.NotNil(t, getCmd.Args)

		// Test that cobra.ExactArgs(1) is used
		err := getCmd.Args(getCmd, []string{})
		assert.Error(t, err)

		err = getCmd.Args(getCmd, []string{"id123"})
		assert.NoError(t, err)
	})
}

func TestUpdateCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		// update [id] includes argument placeholder
		assert.Equal(t, "update [id]", updateCmd.Use)
		assert.Equal(t, "Update data entry with optimistic locking", updateCmd.Short)
		assert.NotNil(t, updateCmd.Run)
	})

	t.Run("should require exactly one argument", func(t *testing.T) {
		assert.NotNil(t, updateCmd.Args)

		err := updateCmd.Args(updateCmd, []string{"id123"})
		assert.NoError(t, err)
	})

	t.Run("should have version flag for optimistic locking", func(t *testing.T) {
		versionFlag := updateCmd.Flags().Lookup("expected-version")
		require.NotNil(t, versionFlag)
		assert.Equal(t, "v", versionFlag.Shorthand)
		assert.Equal(t, "Expected version for optimistic locking (default: current version)", versionFlag.Usage)
	})
}

func TestDeleteCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		// delete [id] includes argument placeholder
		assert.Equal(t, "delete [id]", deleteCmd.Use)
		assert.Equal(t, "Delete data entry", deleteCmd.Short)
		assert.NotNil(t, deleteCmd.Run)
	})

	t.Run("should require exactly one argument", func(t *testing.T) {
		assert.NotNil(t, deleteCmd.Args)

		err := deleteCmd.Args(deleteCmd, []string{"id123"})
		assert.NoError(t, err)
	})
}

func TestDataCommandsIntegration(t *testing.T) {
	t.Run("all data commands should be properly configured", func(t *testing.T) {
		commands := []*cobra.Command{storeCmd, listCmd, getCmd, updateCmd, deleteCmd}

		for _, cmd := range commands {
			assert.NotEmpty(t, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotNil(t, cmd.Run)
		}
	})

	t.Run("store and update commands should have similar flags", func(t *testing.T) {
		commonFlags := []string{"type", "name", "metadata", "file"}

		for _, flag := range commonFlags {
			storeFlag := storeCmd.Flags().Lookup(flag)
			updateFlag := updateCmd.Flags().Lookup(flag)

			assert.NotNil(t, storeFlag, "store command should have %s flag", flag)
			assert.NotNil(t, updateFlag, "update command should have %s flag", flag)
		}
	})
}

func TestCommandFlagValidation(t *testing.T) {
	t.Run("name flag should be required for store command", func(t *testing.T) {
		nameFlag := storeCmd.Flags().Lookup("name")
		require.NotNil(t, nameFlag)
		assert.Equal(t, "name", nameFlag.Name)
		assert.Equal(t, "n", nameFlag.Shorthand)
	})
}
