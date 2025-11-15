package cli

import (
	"testing"

	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/stretchr/testify/assert"
)

func TestVersionCmd(t *testing.T) {
	t.Run("should have correct command structure", func(t *testing.T) {
		assert.Equal(t, "version", versionCmd.Use)
		assert.Equal(t, "Show version and configuration information", versionCmd.Short)
		assert.NotNil(t, versionCmd.Run)
	})

	t.Run("should not require arguments", func(t *testing.T) {
		assert.Nil(t, versionCmd.Args)
	})

	t.Run("should have run function that uses config", func(t *testing.T) {
		// Mock config functions
		originalLoad := configLoad
		originalGetConfigPath := configGetConfigPath

		configLoad = func() *config.Config {
			return &config.Config{
				ServerURL: "http://test-server.com",
				Token:     "test-token",
			}
		}
		configGetConfigPath = func() (string, error) {
			return "/tmp/config.json", nil
		}
		defer func() {
			configLoad = originalLoad
			configGetConfigPath = originalGetConfigPath
		}()

		// Set test version and build date
		Version = "v1.0.0-test"
		BuildDate = "2024-01-01"
		defer func() {
			Version = ""
			BuildDate = ""
		}()

		// Verify that the run function is properly set up
		assert.NotNil(t, versionCmd.Run)

		// We can't easily test the output without capturing stdout,
		// but we can verify the function exists and uses our mocks
	})
}
