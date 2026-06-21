package datastores

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/yca-software/2chi-go-api/internals/config"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_postgresql "github.com/yca-software/2chi-go-postgresql"
	chi_redis "github.com/yca-software/2chi-go-redis"
	"golang.org/x/sync/errgroup"
)

type Datastores struct {
	Postgres       *chi_postgresql.PostgreSQL
	RedisSession   *chi_redis.Redis
	RedisRateLimit *chi_redis.Redis
}

func New(cfg *config.Config, logger chi_logger.Logger) (*Datastores, error) {
	ds := &Datastores{}

	var connGroup errgroup.Group
	connGroup.Go(func() error {
		postgres, err := chi_postgresql.NewPostgreSQL(chi_postgresql.PostgreSQLClientConfig{
			DSN:             cfg.Postgres.DSN,
			MaxOpenConns:    cfg.Postgres.MaxOpenConns,
			MaxIdleConns:    cfg.Postgres.MaxIdleConns,
			ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetime) * time.Second,
			ConnMaxIdleTime: time.Duration(cfg.Postgres.ConnMaxIdleTime) * time.Second,
			PingTimeout:     time.Duration(cfg.Postgres.PingTimeout) * time.Second,
		})

		if err := goose.Up(postgres.GetClient().(*sqlx.DB).DB, cfg.App.MigrationsPath); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		ds.Postgres = postgres
		return err
	})

	connGroup.Go(func() error {
		redisSession, err := chi_redis.NewRedis(chi_redis.RedisClientConfig{
			DSN: cfg.Redis.SessionDSN,
		})
		ds.RedisSession = redisSession
		return err
	})

	connGroup.Go(func() error {
		redisRateLimit, err := chi_redis.NewRedis(chi_redis.RedisClientConfig{
			DSN: cfg.Redis.RateLimitDSN,
		})
		ds.RedisRateLimit = redisRateLimit
		return err
	})

	if err := connGroup.Wait(); err != nil {
		ds.Close()
		return nil, err
	}

	return ds, nil
}

// Close releases all initialized datastore connections in parallel.
func (ds *Datastores) Close() {
	if ds == nil {
		return
	}
	var g errgroup.Group
	if ds.Postgres != nil {
		g.Go(func() error {
			ds.Postgres.Cleanup()
			return nil
		})
	}
	if ds.RedisSession != nil {
		g.Go(func() error {
			ds.RedisSession.Cleanup()
			return nil
		})
	}
	if ds.RedisRateLimit != nil {
		g.Go(func() error {
			ds.RedisRateLimit.Cleanup()
			return nil
		})
	}
	_ = g.Wait()
}
