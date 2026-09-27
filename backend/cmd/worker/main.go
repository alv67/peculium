package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/config"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/service"
	"github.com/alv67/peculium/internal/series"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer dbPool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	var budget price.RateBudget = price.NoopBudget{}
	var cacheClient *redis.Client
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("redis not available, continuing without rate budget")
	} else {
		budget = price.NewRedisBudget(rdb, int64(cfg.YahooGlobalRate), cfg.YahooGlobalWindow)
		cacheClient = rdb
	}

	repos := repository.New(dbPool, repository.NewLookupCache(cacheClient))
	c := cache.New(cacheClient)
	healthSvc := service.NewHealthService(repos)

	fetcher := price.NewYahooFetcher(repos, cfg.PriceFetchInterval,
		price.WithMinInterval(cfg.YahooMinInterval),
		price.WithRateBudget(budget),
		price.WithHealthRecorder(healthSvc),
	)

	log.Info().Dur("interval", cfg.PriceFetchInterval).Msg("price worker started")

	ticker := time.NewTicker(cfg.PriceFetchInterval)
	defer ticker.Stop()

	// The backend applies migrations on startup; wait for the series tables so
	// the initial recompute does not race them.
	schemaCtx, schemaCancel := context.WithTimeout(ctx, 60*time.Second)
	if err := series.WaitForSchema(schemaCtx, repos); err != nil {
		log.Warn().Err(err).Msg("series schema not ready, skipping initial recompute")
	}
	schemaCancel()

	// Run once immediately
	if err := fetcher.FetchAll(ctx); err != nil {
		log.Warn().Err(err).Msg("initial price fetch failed")
	} else if err := series.RecomputeAll(ctx, repos); err != nil {
		log.Warn().Err(err).Msg("initial series recompute failed")
	} else if _, err := c.Bump(ctx); err != nil {
		log.Warn().Err(err).Msg("cache rev bump failed")
	}

	for {
		select {
		case <-ticker.C:
			log.Info().Msg("fetching prices...")
			if err := fetcher.FetchAll(ctx); err != nil {
				log.Warn().Err(err).Msg("price fetch failed")
			} else if err := series.RecomputeAll(ctx, repos); err != nil {
				log.Warn().Err(err).Msg("series recompute failed")
			} else if _, err := c.Bump(ctx); err != nil {
				log.Warn().Err(err).Msg("cache rev bump failed")
			}
		case <-ctx.Done():
			log.Info().Msg("worker shutting down")
			return
		case <-func() chan os.Signal {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			return c
		}():
			log.Info().Msg("worker shutting down")
			return
		}
	}
}
