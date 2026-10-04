package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/config"
	"github.com/alv67/peculium/internal/handler"
	appmw "github.com/alv67/peculium/internal/middleware"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/series"
	"github.com/alv67/peculium/internal/service"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg := config.Load()

	zerolog.SetGlobalLevel(parseLogLevel(cfg.LogLevel))

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations(cfg)
		return
	}

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
		log.Warn().Err(err).Msg("redis not available, continuing without cache")
	} else {
		budget = price.NewRedisBudget(rdb, int64(cfg.YahooGlobalRate), cfg.YahooGlobalWindow)
		cacheClient = rdb
	}

	jwtAuth := auth.NewJWTAuth(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	repos := repository.New(dbPool, repository.NewLookupCache(cacheClient))
	c := cache.New(cacheClient)
	healthSvc := service.NewHealthService(repos)
	fetcher := price.NewYahooFetcher(repos, cfg.PriceFetchInterval,
		price.WithMinInterval(cfg.YahooMinInterval),
		price.WithRateBudget(budget),
		price.WithHealthRecorder(healthSvc),
	)
	svc := service.New(repos, jwtAuth, fetcher, price.NewJustETFFetcher(cfg.PythonServiceURL), cfg.LookupCacheTTL, cfg.ExposureCacheTTL, c, cfg.SeriesMaxPoints, cfg.StalePriceDays, healthSvc,
		service.NewExecDBMaintainer(service.DBConnection{
			Host:     cfg.DBHost,
			Port:     cfg.DBPort,
			User:     cfg.DBUser,
			Name:     cfg.DBName,
			Password: cfg.DBPassword,
			SSLMode:  cfg.DBSSLMode,
		}))

	h := handler.New(svc, jwtAuth)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	setupRoutes(r, h, jwtAuth, repos.User, cfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := series.RecomputeAll(bgCtx, repos); err != nil {
			log.Warn().Err(err).Msg("series backfill failed")
			return
		}
		log.Info().Msg("series backfill complete")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server stopped")
}

func runMigrations(cfg *config.Config) {
	m, err := migrate.New(
		"file://migrations",
		cfg.DSN(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("migration init failed")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("migration failed")
	}

	log.Info().Msg("migrations applied successfully")
}

func setupRoutes(r chi.Router, h *handler.Handler, jwtAuth *auth.JWTAuth, users appmw.UserLoader, cfg *config.Config) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", versionHandler(cfg))

		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.RefreshToken)

		r.Group(func(r chi.Router) {
			r.Use(jwtAuth.Middleware)
			r.Use(appmw.RequireActiveUser(users))

			r.Route("/admin", func(r chi.Router) {
				r.Use(appmw.RequireAdmin)
				r.Get("/users", h.ListAdminUsers)
				r.Patch("/users/{id}", h.UpdateAdminUser)
				r.Post("/users/{id}/reset-password", h.AdminResetPassword)
				r.Get("/settings", h.GetAdminSettings)
				r.Patch("/settings", h.UpdateAdminSettings)
				r.Get("/db/backup", h.AdminDBBackup)
				r.Post("/db/restore", h.AdminDBRestore)
			})

			r.Get("/users/me", h.GetCurrentUser)
			r.Patch("/users/me", h.UpdateCurrentUser)
			r.Post("/users/me/password", h.ChangePassword)

			r.Get("/assets", h.ListAssets)
			r.Get("/assets/search", h.SearchAssets)
			r.Get("/assets/lookup", h.LookupAsset)
			r.Get("/assets/meta", h.GetAssetMeta)
			r.Get("/assets/{id}", h.GetAsset)
			r.Patch("/assets/{id}", h.UpdateAsset)
			r.Get("/assets/{id}/quote", h.GetAssetQuote)
			r.Get("/assets/{id}/splits", h.AssetSplits)
			r.Post("/assets/{id}/fetch-profile", h.FetchAssetProfile)
			r.Get("/assets/{id}/exposure", h.GetAssetExposure)
			r.Put("/assets/{id}/exposure", h.SaveAssetExposure)
			r.Post("/assets/{id}/exposure/derive", h.DeriveAssetRegions)
			r.Post("/assets/{id}/fetch-exposure", h.FetchAssetExposure)
			r.Post("/assets/{id}/fetch-etf-exposure", h.FetchETFExposure)
			r.Post("/assets/{id}/fetch-morningstar-exposure", h.FetchMorningstarExposure)
			r.Post("/assets/{id}/backfill-history", h.BackfillAssetHistory)
			r.Post("/assets", h.CreateAsset)
			r.Post("/assets/sync", h.SyncAssets)
			r.Post("/assets/backfill-meta", h.BackfillAssetMeta)
			r.Delete("/assets/{id}", h.DeleteAsset)

			r.Get("/portfolios", h.ListPortfolios)
			r.Post("/portfolios", h.CreatePortfolio)
			r.Post("/portfolios/import", h.ImportPortfolio)
			r.Get("/portfolios/{id}", h.GetPortfolio)
			r.Get("/portfolios/{id}/export", h.ExportPortfolio)
			r.Patch("/portfolios/{id}", h.UpdatePortfolio)
			r.Delete("/portfolios/{id}", h.DeletePortfolio)

			r.Get("/portfolios/{id}/transactions", h.ListTransactions)
			r.Post("/portfolios/{id}/transactions", h.CreateTransaction)
			r.Patch("/transactions/{id}", h.UpdateTransaction)
			r.Delete("/transactions/{id}", h.DeleteTransaction)

			r.Get("/portfolios/{id}/summary", h.GetPortfolioSummary)
			r.Get("/portfolios/{id}/performance", h.GetPortfolioPerformance)
			r.Get("/portfolios/{id}/performance/buckets", h.GetPortfolioPerformanceBuckets)
			r.Get("/portfolios/{id}/allocation", h.GetPortfolioAllocation)
			r.Get("/portfolios/{id}/allocation/class", h.GetPortfolioClassAllocation)
			r.Get("/portfolios/{id}/allocation/geography", h.GetPortfolioGeographyAllocation)
			r.Get("/portfolios/{id}/allocation/sector", h.GetPortfolioSectorAllocation)
			r.Get("/portfolios/{id}/allocation/drill", h.GetPortfolioAllocationDrill)
			r.Get("/portfolios/{id}/roi", h.GetPortfolioROI)
			r.Get("/portfolios/{id}/history", h.GetPortfolioHistory)

			r.Get("/dashboard", h.GetDashboard)
			r.Get("/dashboard/allocation", h.GetDashboardAllocation)
			r.Get("/dashboard/allocation/drill", h.GetDashboardAllocationDrill)
			r.Get("/dashboard/performance", h.GetDashboardPerformance)

			r.Get("/backup", h.DownloadBackup)
			r.Post("/backup/restore", h.RestoreUserBackup)

			r.Get("/settings/currencies", h.ListCurrencies)
			r.Post("/settings/currencies", h.CreateCurrency)
			r.Delete("/settings/currencies/{code}", h.DeleteCurrency)

			r.Get("/prices/{assetID}", h.GetPrices)
			r.Post("/prices/refresh", h.RefreshPrices)
			r.Get("/health/prices", h.GetPriceHealth)

			r.Get("/jobs", h.ListJobs)
			r.Get("/jobs/{id}", h.GetJob)
		})
	})
}

type versionResponse struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
}

func versionHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(versionResponse{cfg.Version, cfg.Commit, cfg.BuiltAt})
	}
}

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
