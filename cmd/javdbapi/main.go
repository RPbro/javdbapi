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
		&cli.StringFlag{Name: "output", Usage: outputModeSpec.usage()},
		&cli.StringFlag{Name: "output-dir", Value: "./output", Usage: "directory for output files"},
		&cli.DurationFlag{Name: "stale-after", Value: 24 * time.Hour, Usage: "skip fetch when cached file is newer than duration, e.g. 24h, 48h"},
		&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "HTTP request timeout"},
		&cli.StringFlag{Name: "proxy-url", Usage: "HTTP/SOCKS5 proxy URL, e.g. http://127.0.0.1:7890"},
		&cli.StringFlag{Name: "base-url", Value: "https://javdb.com", Usage: "override base URL"},
		&cli.StringFlag{Name: "user-agent", Usage: "custom User-Agent header"},
		&cli.BoolFlag{Name: "debug", Usage: "enable debug logging to stderr"},
		&cli.BoolFlag{Name: "fail-fast", Usage: "stop on first error"},
		&cli.DurationFlag{Name: "delay", Value: 1 * time.Second, Usage: "delay between requests"},
	}
}

func listFlags() []cli.Flag {
	return append(sharedFlags(),
		&cli.IntFlag{Name: "page", Value: 1},
		&cli.IntFlag{Name: "max-pages", Value: 1},
	)
}

func sharedOptionsFromCommand(cmd *cli.Command, stdout io.Writer, stderr io.Writer) (cliapp.SharedOptions, error) {
	outputMode, err := parseOutputMode(cmd.String("output"))
	if err != nil {
		return cliapp.SharedOptions{}, err
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: func() slog.Level {
			if cmd.Bool("debug") {
				return slog.LevelDebug
			}
			return slog.LevelInfo
		}(),
	}))

	return cliapp.SharedOptions{
		OutputMode: outputMode,
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
	}, nil
}

func newListCommand(
	name string,
	usage string,
	usageText string,
	extra []cli.Flag,
	ex executor,
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
			shared, err := sharedOptionsFromCommand(cmd, stdout, stderr)
			if err != nil {
				return err
			}
			req.Shared = shared
			summary, err := ex.RunList(ctx, req)
			logSummary(req.Shared.Logger, summary, err)
			return err
		},
	}
}

func newSearchCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("search", "search videos by keyword", "javdbapi search --keyword VR", []cli.Flag{
		&cli.StringFlag{Name: "keyword", Required: true, Usage: "search keyword"},
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
	return newListCommand("home", "browse home page listings", "javdbapi home --type censored --filter all --sort publish", []cli.Flag{
		newStringFlag("type", homeTypeSpec, false),
		newStringFlag("filter", homeFilterSpec, false),
		newStringFlag("sort", homeSortSpec, false),
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		homeType, err := parseHomeType(cmd.String("type"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		homeFilter, err := parseHomeFilter(cmd.String("filter"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		homeSort, err := parseHomeSort(cmd.String("sort"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandHome,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Home: &javdbapi.HomeQuery{
				Type:   homeType,
				Filter: homeFilter,
				Sort:   homeSort,
			},
		}, nil
	})
}

func newMakerCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("maker", "list videos from a maker", "javdbapi maker --id 7R --filter playable", []cli.Flag{
		&cli.StringFlag{Name: "id", Required: true, Usage: "maker ID"},
		newStringFlag("filter", makerFilterSpec, false),
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		filter, err := parseMakerFilter(cmd.String("filter"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandMaker,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Maker: &javdbapi.MakerQuery{
				MakerID: cmd.String("id"),
				Filter:  filter,
			},
		}, nil
	})
}

func newActorCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("actor", "list videos from an actor", "javdbapi actor --id neRNX --filter cnsub,download", []cli.Flag{
		&cli.StringFlag{Name: "id", Required: true, Usage: "actor ID"},
		newStringFlag("filter", actorFilterSpec, false),
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		filters, err := parseActorFilters(cmd.String("filter"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandActor,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Actor: &javdbapi.ActorQuery{
				ActorID: cmd.String("id"),
				Filters: filters,
			},
		}, nil
	})
}

func newRankingCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return newListCommand("ranking", "fetch ranked videos", "javdbapi ranking --period weekly --type censored", []cli.Flag{
		newStringFlag("period", rankingPeriodSpec, true),
		newStringFlag("type", rankingTypeSpec, true),
	}, ex, stdout, stderr, func(cmd *cli.Command) (cliapp.ListRequest, error) {
		period, err := parseRankingPeriod(cmd.String("period"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		rankingType, err := parseRankingType(cmd.String("type"))
		if err != nil {
			return cliapp.ListRequest{}, err
		}
		return cliapp.ListRequest{
			Command:  cliapp.CommandRanking,
			Page:     cmd.Int("page"),
			MaxPages: cmd.Int("max-pages"),
			Ranking: &javdbapi.RankingQuery{
				Period: period,
				Type:   rankingType,
			},
		}, nil
	})
}

func newVideoCommand(ex executor, stdout io.Writer, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "video",
		Usage:     "fetch full video detail",
		UsageText: "javdbapi video --path /v/ZNdEbV",
		Flags: append(sharedFlags(),
			&cli.StringFlag{Name: "path", Usage: "video path, e.g. /v/ZNdEbV"},
			&cli.StringFlag{Name: "url", Usage: "full video URL; host must match --base-url"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			shared, err := sharedOptionsFromCommand(cmd, stdout, stderr)
			if err != nil {
				return err
			}
			req := cliapp.VideoRequest{
				Shared: shared,
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
