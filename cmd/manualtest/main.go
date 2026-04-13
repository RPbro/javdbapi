package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
)

type endpointName string

const (
	endpointHome    endpointName = "home"
	endpointSearch  endpointName = "search"
	endpointMaker   endpointName = "maker"
	endpointActor   endpointName = "actor"
	endpointRanking endpointName = "ranking"
	endpointVideo   endpointName = "video"
)

var endpointOrder = []endpointName{
	endpointHome,
	endpointSearch,
	endpointMaker,
	endpointActor,
	endpointRanking,
	endpointVideo,
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	onlyFlag, onlyFlagSet, helpRequested, err := parseManualFlags(args)
	if helpRequested {
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		return 1
	}

	cfg, timeout, err := loadClientConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	client, err := javdbapi.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "new client: %v\n", err)
		return 1
	}

	only := resolveOnlyInput(onlyFlag, onlyFlagSet, os.Getenv("JAVDB_MANUAL_ONLY"))

	selected, err := resolveOnlySelection(only)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve selected endpoints: %v\n", err)
		return 1
	}
	if len(selected) == 0 {
		selected = append([]endpointName(nil), endpointOrder...)
	}

	samples := loadSampleInputFromEnv()

	var hasFailure bool
	for _, ep := range selected {
		err := executeWithRetry(timeout, func(ctx context.Context) error {
			return runEndpoint(ctx, client, ep, samples)
		})
		if err != nil {
			hasFailure = true
			fmt.Printf("FAIL %s: %v\n", ep, err)
			continue
		}
		fmt.Printf("PASS %s\n", ep)
	}

	if hasFailure {
		return 1
	}
	return 0
}

func parseManualFlags(args []string) (string, bool, bool, error) {
	normalizedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--help" {
			normalizedArgs = append(normalizedArgs, "-h")
			continue
		}
		normalizedArgs = append(normalizedArgs, arg)
	}

	fs := flag.NewFlagSet("manualtest", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	var only string
	fs.StringVar(&only, "only", "", "comma-separated endpoints: home,search,maker,actor,ranking,video")

	if err := fs.Parse(normalizedArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", false, true, nil
		}
		return "", false, false, err
	}

	onlySet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "only" {
			onlySet = true
		}
	})
	return only, onlySet, false, nil
}

func resolveOnlyInput(flagValue string, flagProvided bool, envValue string) string {
	if flagProvided {
		return strings.TrimSpace(flagValue)
	}
	return strings.TrimSpace(envValue)
}

func resolveOnlySelection(raw string) ([]endpointName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	allowed := map[endpointName]struct{}{
		endpointHome:    {},
		endpointSearch:  {},
		endpointMaker:   {},
		endpointActor:   {},
		endpointRanking: {},
		endpointVideo:   {},
	}

	parts := strings.Split(trimmed, ",")
	selected := make([]endpointName, 0, len(parts))
	seen := make(map[endpointName]struct{}, len(parts))
	for _, part := range parts {
		name := endpointName(strings.TrimSpace(part))
		if name == "" {
			return nil, fmt.Errorf("empty endpoint in selection %q", raw)
		}
		if _, ok := allowed[name]; !ok {
			return nil, fmt.Errorf("unknown endpoint %q", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		selected = append(selected, name)
	}
	return selected, nil
}

type sampleInput struct {
	keyword   string
	makerID   string
	actorID   string
	videoPath string
}

func loadSampleInputFromEnv() sampleInput {
	samples := sampleInput{
		keyword:   strings.TrimSpace(os.Getenv("JAVDB_SAMPLE_KEYWORD")),
		makerID:   strings.TrimSpace(os.Getenv("JAVDB_SAMPLE_MAKER_ID")),
		actorID:   strings.TrimSpace(os.Getenv("JAVDB_SAMPLE_ACTOR_ID")),
		videoPath: strings.TrimSpace(os.Getenv("JAVDB_SAMPLE_VIDEO_PATH")),
	}
	if samples.keyword == "" {
		samples.keyword = "VR"
	}
	if samples.makerID == "" {
		samples.makerID = "7R"
	}
	if samples.actorID == "" {
		samples.actorID = "neRNX"
	}
	if samples.videoPath == "" {
		samples.videoPath = "/v/ZNdEbV"
	}
	return samples
}

func runEndpoint(ctx context.Context, client *javdbapi.Client, endpoint endpointName, samples sampleInput) error {
	switch endpoint {
	case endpointHome:
		videos, err := client.Home(ctx, javdbapi.HomeQuery{Page: 1})
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			return fmt.Errorf("empty result")
		}
		return nil
	case endpointSearch:
		videos, err := client.Search(ctx, javdbapi.SearchQuery{
			Keyword: samples.keyword,
			Page:    1,
		})
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			return fmt.Errorf("empty result")
		}
		return nil
	case endpointMaker:
		if samples.makerID == "" {
			return fmt.Errorf("missing JAVDB_SAMPLE_MAKER_ID")
		}
		videos, err := client.Maker(ctx, javdbapi.MakerQuery{
			MakerID: samples.makerID,
			Page:    1,
		})
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			return fmt.Errorf("empty result")
		}
		return nil
	case endpointActor:
		if samples.actorID == "" {
			return fmt.Errorf("missing JAVDB_SAMPLE_ACTOR_ID")
		}
		videos, err := client.Actor(ctx, javdbapi.ActorQuery{
			ActorID: samples.actorID,
			Page:    1,
		})
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			return fmt.Errorf("empty result")
		}
		return nil
	case endpointRanking:
		videos, err := client.Ranking(ctx, javdbapi.RankingQuery{
			Period: javdbapi.RankingPeriodWeekly,
			Type:   javdbapi.RankingTypeCensored,
			Page:   1,
		})
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			return fmt.Errorf("empty result")
		}
		return nil
	case endpointVideo:
		if samples.videoPath == "" {
			return fmt.Errorf("missing JAVDB_SAMPLE_VIDEO_PATH")
		}
		video, err := client.Video(ctx, javdbapi.VideoQuery{Path: samples.videoPath})
		if err != nil {
			return err
		}
		if video == nil {
			return fmt.Errorf("empty result")
		}
		return nil
	default:
		return fmt.Errorf("unsupported endpoint %q", endpoint)
	}
}

func executeWithRetry(timeout time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := fn(ctx)
	if err == nil {
		return nil
	}
	if !isTransientNetworkError(err) {
		return err
	}

	retryDelay := 200 * time.Millisecond
	timer := time.NewTimer(retryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	return fn(ctx)
}

func isTransientNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	transientHints := []string{
		"timeout",
		"connection reset",
		"broken pipe",
		"eof",
		"temporary",
	}
	for _, hint := range transientHints {
		if strings.Contains(msg, hint) {
			return true
		}
	}
	return false
}

func loadClientConfigFromEnv() (javdbapi.Config, time.Duration, error) {
	timeout := 30 * time.Second
	if raw := strings.TrimSpace(os.Getenv("JAVDB_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return javdbapi.Config{}, 0, fmt.Errorf("parse JAVDB_TIMEOUT: %w", err)
		}
		if parsed <= 0 {
			return javdbapi.Config{}, 0, fmt.Errorf("JAVDB_TIMEOUT must be positive")
		}
		timeout = parsed
	}

	cfg := javdbapi.Config{
		BaseURL:   strings.TrimSpace(os.Getenv("JAVDB_BASE_URL")),
		ProxyURL:  strings.TrimSpace(os.Getenv("JAVDB_PROXY_URL")),
		Timeout:   timeout,
		UserAgent: strings.TrimSpace(os.Getenv("JAVDB_USER_AGENT")),
	}

	return cfg, timeout, nil
}
