package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	cli "github.com/urfave/cli/v3"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/cliapp"
	"github.com/RPbro/javdbapi/internal/clioutput"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// fetcherBuilder constructs the SDK Client for a single command invocation.
// cliapp only ever talks to a Fetcher, so this is the sole seam where
// javdbapi.ClientConfig gets built from CLI flags.
type fetcherBuilder func(cmd *cli.Command, logger *slog.Logger) (cliapp.Fetcher, error)

func formatVersionLine() string {
	return fmt.Sprintf("javdbapi %s (commit %s, built %s)", version, commit, date)
}

func printVersionLine(w io.Writer) error {
	_, err := fmt.Fprintln(w, formatVersionLine())
	return err
}

func newVersionCommand(stdout io.Writer) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print build version information",
		Action: func(context.Context, *cli.Command) error {
			return printVersionLine(stdout)
		},
	}
}

func init() {
	cli.VersionPrinter = func(cmd *cli.Command) {
		_, _ = fmt.Fprintln(cmd.Root().Writer, cmd.Root().Version)
	}
}

func newCommand(buildFetcher fetcherBuilder, stdout io.Writer, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "javdbapi",
		Usage:     "official CLI for javdbapi scraping workflows",
		Version:   formatVersionLine(),
		Writer:    stdout,
		ErrWriter: stderr,
		Commands: []*cli.Command{
			newVersionCommand(stdout),
			newSearchCommand(buildFetcher, stdout, stderr),
			newHomeCommand(buildFetcher, stdout, stderr),
			newMakerCommand(buildFetcher, stdout, stderr),
			newActorCommand(buildFetcher, stdout, stderr),
			newRankingCommand(buildFetcher, stdout, stderr),
			newVideoCommand(buildFetcher, stdout, stderr),
		},
	}
}

func newListCommand(
	name string,
	usage string,
	usageText string,
	extra []cli.Flag,
	buildFetcher fetcherBuilder,
	stdout io.Writer,
	stderr io.Writer,
	build func(*cli.Command) (cliapp.ListRequest, error),
) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		UsageText: usageText,
		Flags:     append(listFlags(), extra...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			req, err := build(cmd)
			if err != nil {
				return err
			}
			logger := loggerFromCommand(cmd, stderr)
			shared, err := sharedOptionsFromCommand(cmd, stdout, logger)
			if err != nil {
				return err
			}
			req.Shared = shared
			req.SummaryOnly = cmd.Bool("summary-only")

			fetcher, err := buildFetcher(cmd, logger)
			if err != nil {
				return err
			}
			store := clioutput.NewStore(shared.OutputDir, time.Now)

			summary, err := cliapp.RunListCommand(ctx, fetcher, store, req)
			logSummary(shared.Logger, summary, err)
			return err
		},
	}
}

// buildRealFetcher constructs one javdbapi.Client per command invocation. The
// same logger instance is passed to both cliapp.SharedOptions and
// javdbapi.ClientConfig, so a single --debug flag controls CLI and SDK log
// verbosity together.
func buildRealFetcher(cmd *cli.Command, logger *slog.Logger) (cliapp.Fetcher, error) {
	client, err := javdbapi.NewClient(javdbapi.ClientConfig{
		BaseURL:   cmd.String("base-url"),
		UserAgent: cmd.String("user-agent"),
		HTTP: javdbapi.HTTPConfig{
			Timeout:  cmd.Duration("timeout"),
			ProxyURL: cmd.String("proxy-url"),
		},
		RateLimit: javdbapi.RateLimitPolicy{
			RequestsPerSecond: cmd.Float64("rate"),
			Burst:             cmd.Int("burst"),
		},
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}
	return cliapp.ClientFetcher{Client: client}, nil
}

func logSummary(logger *slog.Logger, s cliapp.Summary, err error) {
	if err != nil && summaryIsZero(s) {
		return
	}
	fields := []any{
		"pages_scanned", s.PagesScanned,
		"candidates", s.Candidates,
		"deduplicated", s.Deduplicated,
		"fetched", s.Fetched,
		"skipped_fresh", s.SkippedFresh,
		"failed", s.Failed,
		"partial_failed", s.PartialFailed,
		"summaries_output", s.SummariesOutput,
	}
	if err != nil {
		logger.Warn("finished with errors", fields...)
		return
	}
	logger.Info("done", fields...)
}

func summaryIsZero(s cliapp.Summary) bool {
	return s == (cliapp.Summary{})
}
