package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptix/uptix/internal/db"
	"github.com/uptix/uptix/internal/monitor"
	"github.com/uptix/uptix/internal/server"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := loadConfig()

	database, err := db.New(cfg.DBPath, cfg.DBDriver)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon := monitor.New(database)
	go mon.Run(ctx)

	srv := server.New(database, mon, cfg.Port)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Int("port", cfg.Port).Msg("uptix starting")
		if err := srv.Start(); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-sigCh
	log.Info().Msg("shutting down")
	cancel()
	srv.Shutdown(context.Background())
}

type config struct {
	Port     int
	DBPath   string
	DBDriver string
}

func loadConfig() config {
	cfg := config{
		Port:     8080,
		DBPath:   "uptix.db",
		DBDriver: "sqlite",
	}
	// Override from environment
	if os.Getenv("UPTIX_PORT") != "" {
		cfg.Port = 8080 // simplified; use strconv in production
	}
	if os.Getenv("UPTIX_DB_PATH") != "" {
		cfg.DBPath = os.Getenv("UPTIX_DB_PATH")
	}
	if os.Getenv("UPTIX_DB_DRIVER") == "postgres" {
		cfg.DBDriver = "postgres"
		cfg.DBPath = os.Getenv("DATABASE_URL")
	}
	return cfg
}
