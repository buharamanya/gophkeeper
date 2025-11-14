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
	// Убираем password из флагов
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

func init() {
	registerCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
	// Убираем флаг для пароля

	loginCmd.Flags().StringVarP(&login, "login", "l", "", "User login")
	// Убираем флаг для пароля
}
