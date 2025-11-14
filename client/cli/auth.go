package cli

import (
	"fmt"
	"time"

	"github.com/buharamanya/gophkeeper/client/api"
	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register [login] [password]",
	Short: "Register new user",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClientWithConfig(api.ClientConfig{
			BaseURL:    cfg.ServerURL,
			Timeout:    time.Duration(cfg.Timeout) * time.Second,
			SkipVerify: cfg.SkipVerify,
		})

		// Проверка соединения
		if err := client.HealthCheck(); err != nil {
			fmt.Printf("Server connection failed: %v\n", err)
			return
		}

		login := args[0]
		password := args[1]

		resp, err := client.Register(login, password)
		if err != nil {
			fmt.Printf("Registration failed: %v\n", err)
			return
		}

		fmt.Printf("User registered successfully. User ID: %s\n", resp.UserID)
		fmt.Printf("Auth token: %s\n", resp.Token)
	},
}

var loginCmd = &cobra.Command{
	Use:   "login [login] [password]",
	Short: "Login user",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClientWithConfig(api.ClientConfig{
			BaseURL:    cfg.ServerURL,
			Timeout:    time.Duration(cfg.Timeout) * time.Second,
			SkipVerify: cfg.SkipVerify,
		})

		// Проверка соединения
		if err := client.HealthCheck(); err != nil {
			fmt.Printf("Server connection failed: %v\n", err)
			return
		}

		login := args[0]
		password := args[1]

		resp, err := client.Login(login, password)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		fmt.Printf("Login successful. User ID: %s\n", resp.UserID)
		fmt.Printf("Auth token: %s\n", resp.Token)
	},
}
