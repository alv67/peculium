package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/secrets"
)

func main() {
	healthMode := flag.Bool("health", false, "check that every secret file exists and exit")
	flag.Parse()

	dir := os.Getenv("PECULIUM_SECRETS_DIR")
	if dir == "" {
		dir = "/run/peculium"
	}

	if *healthMode {
		if err := secrets.Health(dir); err != nil {
			log.Error().Err(err).Msg("secrets health check failed")
			os.Exit(1)
		}
		os.Exit(0)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	generated, err := secrets.Ensure(dir)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to ensure secrets")
	}

	reused := make([]string, 0, len(secrets.Known()))
	for _, s := range secrets.Known() {
		if !contains(generated, s.Name) {
			reused = append(reused, s.Name)
		}
	}
	log.Info().Str("dir", dir).Strs("generated", generated).Strs("reused", reused).Msg("secrets ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("secrets-init stopped")
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
