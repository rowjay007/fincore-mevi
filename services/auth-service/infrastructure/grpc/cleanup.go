package grpc

import (
	"context"
	"time"
)

func CleanupRefreshSessions(ctx context.Context, db DB, revokedRetention time.Duration) (expiredDeleted int64, revokedDeleted int64, err error) {
	now := time.Now().UTC()

	res1, err := db.Exec(ctx, `delete from auth_refresh_sessions where expires_at < $1 or absolute_expires_at < $1`, now)
	if err != nil {
		return 0, 0, err
	}

	var res2Rows int64
	if revokedRetention > 0 {
		cutoff := now.Add(-revokedRetention)
		res2, err := db.Exec(ctx, `delete from auth_refresh_sessions where revoked_at is not null and revoked_at < $1`, cutoff)
		if err != nil {
			return res1.RowsAffected(), 0, err
		}
		res2Rows = res2.RowsAffected()
	}

	return res1.RowsAffected(), res2Rows, nil
}
