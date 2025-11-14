package api

import (
	"context"
	"testing"
)

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	// Тестируем добавление и извлечение значений
	ctx = WithUserID(ctx, "test-user-id")
	ctx = WithUserLogin(ctx, "test-user-login")

	userID, ok := GetUserID(ctx)
	if !ok || userID != "test-user-id" {
		t.Errorf("Expected userID 'test-user-id', got '%s'", userID)
	}

	userLogin, ok := GetUserLogin(ctx)
	if !ok || userLogin != "test-user-login" {
		t.Errorf("Expected userLogin 'test-user-login', got '%s'", userLogin)
	}

	// Тестируем отсутствие значений
	emptyCtx := context.Background()
	if _, ok := GetUserID(emptyCtx); ok {
		t.Error("Expected no userID in empty context")
	}

	if _, ok := GetUserLogin(emptyCtx); ok {
		t.Error("Expected no userLogin in empty context")
	}
}
