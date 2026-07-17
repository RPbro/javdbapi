package main

import (
	"io"
	"log/slog"
	"time"

	cli "github.com/urfave/cli/v3"

	"github.com/RPbro/javdbapi/internal/cliapp"
)

func sharedFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "output", Usage: outputModeSpec.usage()},
		&cli.StringFlag{Name: "output-dir", Value: "./output", Usage: "directory for output files"},
		&cli.DurationFlag{Name: "stale-after", Value: 24 * time.Hour, Usage: "skip fetch when cached file is newer than duration, e.g. 24h, 48h"},
		&cli.IntFlag{Name: "concurrency", Value: 2, Usage: "number of videos fetched concurrently (1-16)"},
		&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "HTTP request timeout"},
		&cli.StringFlag{Name: "proxy-url", Usage: "HTTP/SOCKS5 proxy URL, e.g. http://127.0.0.1:7890"},
		&cli.StringFlag{Name: "base-url", Value: "https://javdb.com", Usage: "override base URL"},
		&cli.StringFlag{Name: "user-agent", Usage: "custom User-Agent header"},
		&cli.Float64Flag{Name: "rate", Value: 1, Usage: "requests per second"},
		&cli.IntFlag{Name: "burst", Value: 1, Usage: "rate limiter burst size"},
		&cli.BoolFlag{Name: "debug", Usage: "enable debug logging to stderr"},
		&cli.BoolFlag{Name: "fail-fast", Usage: "stop on first error"},
	}
}

func listFlags() []cli.Flag {
	return append(sharedFlags(),
		&cli.IntFlag{Name: "page", Value: 1},
		&cli.IntFlag{Name: "max-pages", Value: 1},
		&cli.BoolFlag{Name: "summary-only", Usage: "output VideoSummary NDJSON only; skips Detail/Reviews and cache (requires --output console)"},
	)
}

func loggerFromCommand(cmd *cli.Command, stderr io.Writer) *slog.Logger {
	level := slog.LevelInfo
	if cmd.Bool("debug") {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))
}

func sharedOptionsFromCommand(cmd *cli.Command, stdout io.Writer, logger *slog.Logger) (cliapp.SharedOptions, error) {
	outputMode, err := parseOutputMode(cmd.String("output"))
	if err != nil {
		return cliapp.SharedOptions{}, err
	}
	concurrency, err := parseConcurrency(cmd.Int("concurrency"))
	if err != nil {
		return cliapp.SharedOptions{}, err
	}

	return cliapp.SharedOptions{
		OutputMode:  outputMode,
		OutputDir:   cmd.String("output-dir"),
		StaleAfter:  cmd.Duration("stale-after"),
		Concurrency: concurrency,
		FailFast:    cmd.Bool("fail-fast"),
		Stdout:      stdout,
		Logger:      logger,
	}, nil
}
