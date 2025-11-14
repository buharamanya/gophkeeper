package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		assert.Equal(t, "sync", syncCmd.Use)
		assert.Equal(t, "Synchronize data with server", syncCmd.Short)
		assert.NotNil(t, syncCmd.Run)
	})

	t.Run("should have force sync flag", func(t *testing.T) {
		forceFlag := syncCmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)
		assert.Equal(t, "f", forceFlag.Shorthand)
		assert.Equal(t, "Force full synchronization", forceFlag.Usage)
	})

	t.Run("should have conflict resolution flags", func(t *testing.T) {
		resolveAllFlag := syncCmd.Flags().Lookup("resolve-all")
		require.NotNil(t, resolveAllFlag)
		assert.Equal(t, "Automatically resolve all conflicts", resolveAllFlag.Usage)

		clientWinsFlag := syncCmd.Flags().Lookup("client-wins")
		require.NotNil(t, clientWinsFlag)
		assert.Equal(t, "Use client version when resolving conflicts", clientWinsFlag.Usage)
	})
}

func TestSyncStatusCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		assert.Equal(t, "sync-status", syncStatusCmd.Use)
		assert.Equal(t, "Show synchronization status", syncStatusCmd.Short)
		assert.NotNil(t, syncStatusCmd.Run)
	})

	t.Run("should not require arguments", func(t *testing.T) {
		assert.Nil(t, syncStatusCmd.Args)
	})
}

func TestResolveConflictCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		// sync-resolve [conflict-id] includes argument placeholder
		assert.Equal(t, "sync-resolve [conflict-id]", resolveConflictCmd.Use)
		assert.Equal(t, "Resolve synchronization conflict", resolveConflictCmd.Short)
		assert.NotNil(t, resolveConflictCmd.Run)
	})

	t.Run("should require exactly one argument", func(t *testing.T) {
		assert.NotNil(t, resolveConflictCmd.Args)

		err := resolveConflictCmd.Args(resolveConflictCmd, []string{"conflict-123"})
		assert.NoError(t, err)
	})
}

func TestSyncCommandIntegration(t *testing.T) {
	t.Run("all sync commands should be properly configured", func(t *testing.T) {
		commands := []*cobra.Command{syncCmd, syncStatusCmd, resolveConflictCmd}

		for _, cmd := range commands {
			assert.NotEmpty(t, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotNil(t, cmd.Run)
		}
	})
}
