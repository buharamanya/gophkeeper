package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/buharamanya/gophkeeper/client/cli"
)

var (
	version   = "1.0.0"
	buildDate = "2024-12-01"
)

func main() {
	// Создаем контекст с отменой для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Запускаем выполнение команд в отдельной горутине
	done := make(chan error, 1)
	go func() {
		done <- cli.Execute(version, buildDate)
	}()

	// Ожидаем завершение команды или сигнал
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, shutting down...")
		// Можно добавить cleanup для клиента если нужно
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
