package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
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

type executor interface {
	RunList(context.Context, cliapp.ListRequest) (cliapp.Summary, error)
	RunVideo(context.Context, cliapp.VideoRequest) (cliapp.Summary, error)
}

func main() {
	cmd := newCommand(newRealExecutor(os.Stdout, os.Stderr), os.Stdout, os.Stderr)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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

func newCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "javdbapi",
		Usage:     "official CLI for javdbapi scraping workflows",
		Version:   formatVersionLine(),
		Writer:    stdout,
		ErrWriter: stderr,
		Commands: []*cli.Command{
			newVersionCommand(stdout),
			newSearchCommand(ex, stdout, stderr),
			newHomeCommand(ex, stdout, stderr),
			newMakerCommand(ex, stdout, stderr),
			newActorCommand(ex, stdout, stderr),
			newRankingCommand(ex, stdout, stderr),
			newVideoCommand(ex, stdout, stderr),
		},
	}
}

func sharedFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "output", Value: "file"},
		&cli.StringFlag{Name: "output-dir", Value: "./output"},
		&cli.DurationFlag{Name: "stale-after", Value: 24 * time.Hour},
		&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second},
		&cli.StringFlag{Name: "proxy-url"},
		&cli.StringFlag{Name: "base-url", Value: "https://javdb.com"},
		&cli.StringFlag{Name: "user-agent"},
		&cli.BoolFlag{Name: "debug"},
		&cli.BoolFlag{Name: "fail-fast"},
		&cli.DurationFlag{Name: "delay", Value: 1 * time.Second},
	}
}

func listFlags() []cli.Flag {
	return append(sharedFlags(),
		&cli.IntFlag{Name: "page", Value: 1},
		&cli.IntFlag{Name: "max-pages", Value: 1},
	)
}

func sharedOptionsFromCommand(cmd *cli.Command, stdout io.Writer, stderr io.Writer) cliapp.SharedOptions {
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: func() slog.Level {
			if cmd.Bool("debug") {
				return slog.LevelDebug
			}
			return slog.LevelInfo
		}(),
	}))

	return cliapp.SharedOptions{
		OutputMode: cliapp.OutputMode(cmd.String("output")),
		OutputDir:  cmd.String("output-dir"),
		StaleAfter: cmd.Duration("stale-after"),
		Timeout:    cmd.Duration("timeout"),
		BaseURL:    cmd.String("base-url"),
		ProxyURL:   cmd.String("proxy-url"),
		UserAgent:  cmd.String("user-agent"),
		Debug:      cmd.Bool("debug"),
		Stdout:     stdout,
		Logger:     logger,
		FailFast:   cmd.Bool("fail-fast"),
		Delay:      cmd.Duration("delay"),
	}
}

func newListCommand(
	name string,
	extra []cli.Flag,
	ex executor,
	stdout io.Writer,
	stderr io.Writer,
	build func(*cli.Command) (cliapp.ListRequest, error),
) *cli.Command {
	return &cli.Command{
		Name:  name,
		Flags: append(listFlags(), extra...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			req, err := build(cmd)
			if err != nil {
				return err
			}
			req.Shared = sharedOptionsFromCommand(cmd, stdout, stderr)
			summary, err := ex.RunList(ctx, req)
			logSummary(req.Shared.Logger, summary, err)
			return err
		},
	}
}

func newSearchCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("search", []cli.Flag{
		&cli.StringFlag{Name: "keyword", Required: true},
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		return cliapp.ListRequest{
			Command:  cliapp.CommandSearch,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Search:   &javdbapi.SearchQuery{Keyword: cmd.String("keyword")},
		}, nil
	})
}

func newHomeCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("home", []cli.Flag{
		&cli.StringFlag{Name: "type"},
		&cli.StringFlag{Name: "filter"},
		&cli.StringFlag{Name: "sort"},
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		return cliapp.ListRequest{
			Command:  cliapp.CommandHome,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Home: &javdbapi.HomeQuery{
				Type:   javdbapi.HomeType(cmd.String("type")),
				Filter: javdbapi.HomeFilter(cmd.String("filter")),
				Sort:   javdbapi.HomeSort(cmd.String("sort")),
			},
		}, nil
	})
}

func newMakerCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("maker", []cli.Flag{
		&cli.StringFlag{Name: "id", Required: true},
		&cli.StringFlag{Name: "filter"},
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		filter := cmd.String("filter")
		if strings.Contains(filter, ",") {
			return cliapp.ListRequest{}, fmt.Errorf("maker --filter accepts a single value")
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandMaker,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Maker: &javdbapi.MakerQuery{
				MakerID: cmd.String("id"),
				Filter:  javdbapi.MakerFilter(filter),
			},
		}, nil
	})
}

func newActorCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("actor", []cli.Flag{
		&cli.StringFlag{Name: "id", Required: true},
		&cli.StringFlag{Name: "filter"},
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		rawFilters := parseCommaValues(cmd.String("filter"))
		actorFilters := make([]javdbapi.ActorFilter, 0, len(rawFilters))
		for _, filter := range rawFilters {
			actorFilters = append(actorFilters, javdbapi.ActorFilter(filter))
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandActor,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Actor: &javdbapi.ActorQuery{
				ActorID: cmd.String("id"),
				Filters: actorFilters,
			},
		}, nil
	})
}

func newRankingCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("ranking", []cli.Flag{
		&cli.StringFlag{Name: "period", Required: true},
		&cli.StringFlag{Name: "type", Required: true},
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		return cliapp.ListRequest{
			Command:  cliapp.CommandRanking,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Ranking: &javdbapi.RankingQuery{
				Period: javdbapi.RankingPeriod(cmd.String("period")),
				Type:   javdbapi.RankingType(cmd.String("type")),
			},
		}, nil
	})
}

func newVideoCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name: "video",
		Flags: append(sharedFlags(),
			&cli.StringFlag{Name: "path"},
			&cli.StringFlag{Name: "url"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			req := cliapp.VideoRequest{
				Shared: sharedOptionsFromCommand(cmd, stdout, stderr),
				Path:   cmd.String("path"),
				URL:    cmd.String("url"),
			}
			summary, err := ex.RunVideo(ctx, req)
			logSummary(req.Shared.Logger, summary, err)
			return err
		},
	}
}

func parseCommaValues(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// realExecutor constructs a javdbapi.Client per command invocation.
type realExecutor struct {
	stdout io.Writer
	stderr io.Writer
}

func newRealExecutor(stdout io.Writer, stderr io.Writer) executor {
	return &realExecutor{stdout: stdout, stderr: stderr}
}

func (e *realExecutor) RunList(ctx context.Context, req cliapp.ListRequest) (cliapp.Summary, error) {
	client, err := javdbapi.NewClient(javdbapi.Config{
		BaseURL:   req.Shared.BaseURL,
		Timeout:   req.Shared.Timeout,
		ProxyURL:  req.Shared.ProxyURL,
		UserAgent: req.Shared.UserAgent,
		Debug:     req.Shared.Debug,
	})
	if err != nil {
		return cliapp.Summary{}, err
	}

	store := clioutput.NewStore(req.Shared.OutputDir, time.Now)
	return cliapp.RunListCommand(ctx, cliapp.ClientFetcher{Client: client}, store, req)
}

func (e *realExecutor) RunVideo(ctx context.Context, req cliapp.VideoRequest) (cliapp.Summary, error) {
	client, err := javdbapi.NewClient(javdbapi.Config{
		BaseURL:   req.Shared.BaseURL,
		Timeout:   req.Shared.Timeout,
		ProxyURL:  req.Shared.ProxyURL,
		UserAgent: req.Shared.UserAgent,
		Debug:     req.Shared.Debug,
	})
	if err != nil {
		return cliapp.Summary{}, err
	}

	store := clioutput.NewStore(req.Shared.OutputDir, time.Now)
	return cliapp.RunVideoCommand(ctx, cliapp.ClientFetcher{Client: client}, store, req)
}

func logSummary(logger *slog.Logger, s cliapp.Summary, err error) {
	if err != nil && summaryIsZero(s) {
		return
	}
	if err != nil {
		logger.Warn("finished with errors",
			"pages_scanned", s.PagesScanned,
			"candidates", s.Candidates,
			"deduplicated", s.Deduplicated,
			"fetched", s.Fetched,
			"skipped_fresh", s.SkippedFresh,
			"failed", s.Failed,
		)
		return
	}

	logger.Info("done",
		"pages_scanned", s.PagesScanned,
		"candidates", s.Candidates,
		"deduplicated", s.Deduplicated,
		"fetched", s.Fetched,
		"skipped_fresh", s.SkippedFresh,
		"failed", s.Failed,
	)
}

func summaryIsZero(s cliapp.Summary) bool {
	return s == (cliapp.Summary{})
}
