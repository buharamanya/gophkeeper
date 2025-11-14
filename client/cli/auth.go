package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/buharamanya/gophkeeper/client/api"
	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/spf13/cobra"
)

var (
	login    string
	password string
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)

		if login == "" || password == "" {
			login, password = getCredentials()
		}

		resp, err := client.Register(login, password)
		if err != nil {
			fmt.Printf("Registration failed: %v\n", err)
			return
		}

		// Сохраняем токен в конфиг
		cfg.Token = resp.Token
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Failed to save token: %v\n", err)
			return
		}

		fmt.Printf("Registration successful! User ID: %s\n", resp.UserID)
		fmt.Printf("Token saved to config\n")
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)

		if login == "" || password == "" {
			login, password = getCredentials()
		}

		resp, err := client.Login(login, password)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		// Сохраняем токен в конфиг
		cfg.Token = resp.Token
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Failed to save token: %v\n", err)
			return
		}

		fmt.Printf("Login successful! User ID: %s\n", resp.UserID)
		fmt.Printf("Token saved to config\n")
	},
}

// Упрощенная версия без golang.org/x/term
func getCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter login: ")
	loginInput, _ := reader.ReadString('\n')
	loginInput = strings.TrimSpace(loginInput)

	fmt.Print("Enter password: ")
	passwordInput, _ := reader.ReadString('\n')
	passwordInput = strings.TrimSpace(passwordInput)

	return loginInput, passwordInput
}

func init() {
	registerCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
	registerCmd.Flags().StringVarP(&password, "password", "p", "", "User password")

	loginCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "User password")
}
