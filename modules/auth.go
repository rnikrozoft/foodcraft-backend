package main

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

func afterAuthenticateDevice(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, _ *api.Session, _ *api.AuthenticateDeviceRequest) error {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userID == "" {
		return nil
	}
	if err := ensurePlayerStateExists(ctx, nk, userID); err != nil {
		logger.Warn("failed to initialize player state for %s: %v", userID, err)
	}
	return nil
}
