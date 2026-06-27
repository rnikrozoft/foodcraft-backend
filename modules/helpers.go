package main

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcError(message string, code int) error {
	return runtime.NewError(message, code)
}

func mustUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userID == "" {
		return "", rpcError("unauthorized", 16)
	}
	return userID, nil
}

func mustUsername(ctx context.Context) string {
	username, _ := ctx.Value(runtime.RUNTIME_CTX_USERNAME).(string)
	return username
}

func ensureCatalog(logger runtime.Logger) error {
	if catalog != nil {
		return nil
	}
	_, err := loadCatalog()
	if err != nil {
		logger.Error("failed to load catalog: %v", err)
	}
	return err
}
