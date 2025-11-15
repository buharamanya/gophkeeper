package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/buharamanya/gophkeeper/client/api"
	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	login string
)

// getCredentials запрашивает логин и пароль у пользователя
func getCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter login: ")
	loginInput, _ := reader.ReadString('\n')
	loginInput = strings.TrimSpace(loginInput)

	fmt.Print("Enter password: ")
	// Читаем пароль без эха на экране
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Переводим строку после ввода пароля

	var passwordInput string
	if err != nil {
		// В случае ошибки fallback на обычный ввод (менее безопасный)
		fmt.Println("Warning: Cannot read password securely")
		passwordInput, _ = reader.ReadString('\n')
		passwordInput = strings.TrimSpace(passwordInput)
	} else {
		passwordInput = string(passwordBytes)
	}

	return loginInput, passwordInput
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)

		var password string
		if login == "" {
			// Если логин не передан флагом, запрашиваем оба поля интерактивно
			login, password = getCredentials()
		} else {
			// Если логин передан флагом, запрашиваем только пароль
			fmt.Print("Enter password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				return
			}
			password = string(passwordBytes)
		}

		resp, err := client.Register(login, password)
		if err != nil {
			fmt.Printf("Registration failed: %v\n", err)
			return
		}

		// Сохраняем токен в конфиг файл (безопасно)
		if err := config.SaveToken(resp.Token); err != nil {
			fmt.Printf("Failed to save token: %v\n", err)
			return
		}

		fmt.Printf("Registration successful! User ID: %s\n", resp.UserID)
		fmt.Printf("Token saved securely to config file\n")
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)

		var password string
		if login == "" {
			login, password = getCredentials()
		} else {
			fmt.Print("Enter password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				return
			}
			password = string(passwordBytes)
		}

		resp, err := client.Login(login, password)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		// Сохраняем токен в конфиг файл (безопасно)
		if err := config.SaveToken(resp.Token); err != nil {
			fmt.Printf("Failed to save token: %v\n", err)
			return
		}

		fmt.Printf("Login successful! User ID: %s\n", resp.UserID)
		fmt.Printf("Token saved securely to config file\n")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear saved token",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.ClearToken(); err != nil {
			fmt.Printf("Failed to clear token: %v\n", err)
			return
		}
		fmt.Println("Logged out successfully. Token cleared from config file.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()

		configPath, err := config.GetConfigPath()
		if err != nil {
			fmt.Printf("Error getting config path: %v\n", err)
			return
		}

		fmt.Printf("Config file: %s\n", configPath)
		fmt.Printf("Server URL: %s\n", cfg.ServerURL)

		if cfg.Token != "" {
			fmt.Printf("Authentication: ✅ Authenticated\n")
			fmt.Printf("Token: ******** (stored securely)\n")
		} else {
			fmt.Printf("Authentication: ❌ Not authenticated\n")
			fmt.Printf("Use 'gophkeeper login' to authenticate\n")
		}
	},
}

func init() {
	registerCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
	loginCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
}
