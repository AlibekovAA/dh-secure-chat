package cleanup

import (
	"context"
	"expvar"
	"time"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

var (
	refreshTokensCleanupDeleted = expvar.NewInt("refresh_tokens_cleanup_deleted")
	revokedTokensCleanupDeleted = expvar.NewInt("revoked_tokens_cleanup_deleted")
)

type ExpiredDeleter interface {
	DeleteExpired(ctx context.Context) (int64, error)
}

func StartCleanup(ctx context.Context, repo ExpiredDeleter, log *logger.Logger, repoName string, metric *expvar.Int) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := repo.DeleteExpired(ctx)
			if err != nil {
				log.Errorf("%s cleanup failed: %v", repoName, err)
				continue
			}
			if deleted > 0 {
				if metric != nil {
					metric.Add(deleted)
				}
				log.Infof("%s cleanup: deleted %d expired tokens", repoName, deleted)
			}
		}
	}
}

func StartRefreshTokenCleanup(ctx context.Context, repo authrepo.RefreshTokenRepository, log *logger.Logger) {
	StartCleanup(ctx, repo, log, "refresh token", refreshTokensCleanupDeleted)
}

func StartRevokedTokenCleanup(ctx context.Context, repo authrepo.RevokedTokenRepository, log *logger.Logger) {
	StartCleanup(ctx, repo, log, "revoked token", revokedTokensCleanupDeleted)
}
