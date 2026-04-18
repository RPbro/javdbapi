package cliapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/clioutput"
	"github.com/RPbro/javdbapi/internal/web"
)

const rateLimitMaxAttempts = 3

func validateSharedOptions(shared SharedOptions) error {
	switch shared.OutputMode {
	case OutputConsole, OutputFile, OutputBoth:
	default:
		return fmt.Errorf("invalid --output %q: must be one of console, file, both", shared.OutputMode)
	}
	return nil
}

func RunListCommand(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req ListRequest) (Summary, error) {
	if err := validateSharedOptions(req.Shared); err != nil {
		return Summary{}, err
	}
	if req.Page < 1 {
		return Summary{}, fmt.Errorf("invalid --page %d: must be >= 1", req.Page)
	}
	if req.MaxPages < 1 {
		return Summary{}, fmt.Errorf("invalid --max-pages %d: must be >= 1", req.MaxPages)
	}

	summary := Summary{}
	refs := make([]VideoRef, 0)
	seen := make(map[string]struct{})

	for page := req.Page; page < req.Page+req.MaxPages; page++ {
		req.Shared.Logger.Info("fetching list page", "command", req.Command, "page", page)
		videos, err := fetchListPageWithRetry(ctx, fetcher, req, page)
		if err != nil {
			if errors.Is(err, javdbapi.ErrEmptyResult) {
				return processRefs(ctx, fetcher, store, req, refs, summary)
			}
			return summary, err
		}

		summary.PagesScanned++
		summary.Candidates += len(videos)

		for _, video := range videos {
			path := video.ID
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			refs = append(refs, VideoRef{
				Path:  path,
				Title: video.Title,
				Code:  video.Code,
				Page:  page,
			})
		}
	}

	return processRefs(ctx, fetcher, store, req, refs, summary)
}

func fetchListPage(ctx context.Context, fetcher Fetcher, req ListRequest, page int) ([]javdbapi.Video, error) {
	switch req.Command {
	case CommandSearch:
		query := *req.Search
		query.Page = page
		return fetcher.Search(ctx, query)
	case CommandHome:
		query := *req.Home
		query.Page = page
		return fetcher.Home(ctx, query)
	case CommandMaker:
		query := *req.Maker
		query.Page = page
		return fetcher.Maker(ctx, query)
	case CommandActor:
		query := *req.Actor
		query.Page = page
		return fetcher.Actor(ctx, query)
	case CommandRanking:
		query := *req.Ranking
		query.Page = page
		return fetcher.Ranking(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported command %q", req.Command)
	}
}

func processRefs(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req ListRequest, refs []VideoRef, summary Summary) (Summary, error) {
	summary.Deduplicated = len(refs)
	for i, ref := range refs {
		if i > 0 && req.Shared.Delay > 0 {
			select {
			case <-ctx.Done():
				return summary, ctx.Err()
			case <-time.After(req.Shared.Delay):
			}
		}

		source, err := sourceForList(req, ref)
		if err != nil {
			summary.Failed++
			if req.Shared.FailFast {
				return summary, err
			}
			continue
		}

		result, err := processVideoPath(ctx, fetcher, store, req.Shared, ref.Path, []clioutput.Source{source}, false)
		summary.Fetched += result.Fetched
		summary.SkippedFresh += result.SkippedFresh
		summary.Failed += result.Failed

		if result.SkippedFresh > 0 {
			req.Shared.Logger.Debug("skipped fresh", "path", ref.Path)
		} else if result.Failed > 0 {
			req.Shared.Logger.Warn("video failed", "path", ref.Path, "error", err)
		} else {
			req.Shared.Logger.Info("fetched video", "path", ref.Path, "code", ref.Code, "progress", fmt.Sprintf("%d/%d", summary.Fetched+summary.SkippedFresh+summary.Failed, summary.Deduplicated))
		}

		if err != nil && req.Shared.FailFast {
			return summary, err
		}
	}

	if summary.Failed > 0 {
		return summary, fmt.Errorf("list command completed with %d failed videos", summary.Failed)
	}
	return summary, nil
}

func sourceForList(req ListRequest, ref VideoRef) (clioutput.Source, error) {
	switch req.Command {
	case CommandSearch:
		return clioutput.NewSearchSource(req.Search.Keyword, ref.Page), nil
	case CommandHome:
		return clioutput.NewHomeSource(string(req.Home.Type), string(req.Home.Filter), string(req.Home.Sort), ref.Page), nil
	case CommandMaker:
		return clioutput.NewMakerSource(req.Maker.MakerID, string(req.Maker.Filter), ref.Page), nil
	case CommandActor:
		filters := make([]string, 0, len(req.Actor.Filters))
		for _, filter := range req.Actor.Filters {
			filters = append(filters, string(filter))
		}
		return clioutput.NewActorSource(req.Actor.ActorID, filters, ref.Page), nil
	case CommandRanking:
		return clioutput.NewRankingSource(string(req.Ranking.Period), string(req.Ranking.Type), ref.Page), nil
	default:
		return clioutput.Source{}, fmt.Errorf("unsupported command %q", req.Command)
	}
}

func processVideoPath(ctx context.Context, fetcher Fetcher, store *clioutput.Store, shared SharedOptions, path string, sources []clioutput.Source, logFetch bool) (Summary, error) {
	summary := Summary{}
	state, err := loadCacheForMode(store, shared, path)
	if err != nil {
		return summary, err
	}
	if state.Fresh {
		summary.SkippedFresh++
		return summary, nil
	}

	if logFetch {
		shared.Logger.Info("fetching video", "path", path)
	}

	video, err := fetchVideoWithRetry(ctx, fetcher, shared, path)
	if err != nil {
		summary.Failed++
		return summary, err
	}
	if video == nil {
		summary.Failed++
		return summary, fmt.Errorf("video detail returned nil for %s", path)
	}
	if video.ID != "" && video.ID != path {
		summary.Failed++
		return summary, fmt.Errorf("canonical path mismatch: input=%s resolved=%s", path, video.ID)
	}

	pathKey, err := clioutput.PathKeyFromVideoPath(path)
	if err != nil {
		summary.Failed++
		return summary, err
	}

	existingSources := []clioutput.Source(nil)
	if state.Document != nil {
		existingSources = state.Document.Metadata.Sources
	}

	doc := clioutput.Document{
		Metadata: clioutput.Metadata{
			LastUpdated: time.Now().UTC(),
			Path:        path,
			PathKey:     pathKey,
			Sources:     clioutput.MergeSources(existingSources, sources),
		},
		Video: *video,
	}

	if shared.OutputMode == OutputFile || shared.OutputMode == OutputBoth {
		if err := store.WriteFile(doc); err != nil {
			summary.Failed++
			return summary, err
		}
	}
	if shared.OutputMode == OutputConsole || shared.OutputMode == OutputBoth {
		if err := store.WriteJSON(shared.Stdout, doc); err != nil {
			summary.Failed++
			return summary, err
		}
	}

	summary.Fetched++
	return summary, nil
}

func loadCacheForMode(store *clioutput.Store, shared SharedOptions, path string) (clioutput.CacheState, error) {
	if shared.OutputMode == OutputConsole {
		if _, err := os.Stat(shared.OutputDir); errors.Is(err, os.ErrNotExist) {
			filePath, pathErr := store.FilePath(path)
			if pathErr != nil {
				return clioutput.CacheState{}, pathErr
			}
			return clioutput.CacheState{FilePath: filePath}, nil
		}
	}
	return store.Load(path, shared.StaleAfter)
}

func RunVideoCommand(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req VideoRequest) (Summary, error) {
	if err := validateSharedOptions(req.Shared); err != nil {
		return Summary{}, err
	}

	path, err := canonicalVideoPath(req.Shared.BaseURL, req.Path, req.URL)
	if err != nil {
		return Summary{}, err
	}

	return processVideoPath(ctx, fetcher, store, req.Shared, path, []clioutput.Source{
		clioutput.NewVideoSource(path),
	}, true)
}

func canonicalVideoPath(baseURL string, pathValue string, urlValue string) (string, error) {
	switch {
	case pathValue == "" && urlValue == "":
		return "", fmt.Errorf("exactly one of --path or --url is required")
	case pathValue != "" && urlValue != "":
		return "", fmt.Errorf("exactly one of --path or --url is required")
	case pathValue != "":
		return normalizeVideoPath(pathValue)
	default:
		base, err := url.Parse(baseURL)
		if err != nil {
			return "", fmt.Errorf("parse --base-url: %w", err)
		}
		if base.Host == "" {
			return "", fmt.Errorf("invalid --base-url %q", baseURL)
		}

		parsed, err := url.Parse(urlValue)
		if err != nil {
			return "", fmt.Errorf("parse --url: %w", err)
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("invalid --url %q: must be an absolute site video URL", urlValue)
		}
		if !strings.EqualFold(parsed.Host, base.Host) {
			return "", fmt.Errorf("invalid --url host %q: must match --base-url host %q", parsed.Host, base.Host)
		}
		return normalizeVideoPath(parsed.Path)
	}
}

func normalizeVideoPath(raw string) (string, error) {
	key, err := clioutput.PathKeyFromVideoPath(raw)
	if err != nil {
		return "", err
	}
	return "/v/" + key, nil
}

func fetchListPageWithRetry(ctx context.Context, fetcher Fetcher, req ListRequest, page int) ([]javdbapi.Video, error) {
	return withRateLimitRetry(ctx, req.Shared, []any{
		"kind", "list_page",
		"command", req.Command,
		"page", page,
	}, func() ([]javdbapi.Video, error) {
		return fetchListPage(ctx, fetcher, req, page)
	})
}

func fetchVideoWithRetry(ctx context.Context, fetcher Fetcher, shared SharedOptions, path string) (*javdbapi.Video, error) {
	return withRateLimitRetry(ctx, shared, []any{
		"kind", "video",
		"path", path,
	}, func() (*javdbapi.Video, error) {
		return fetcher.Video(ctx, javdbapi.VideoQuery{Path: path})
	})
}

func withRateLimitRetry[T any](ctx context.Context, shared SharedOptions, attrs []any, fn func() (T, error)) (T, error) {
	var zero T

	for attempt := 1; attempt <= rateLimitMaxAttempts; attempt++ {
		value, err := fn()
		if err == nil {
			return value, nil
		}
		if !isRateLimited(err) || attempt == rateLimitMaxAttempts {
			return zero, err
		}

		backoff := retryBackoff(shared.Delay, attempt)
		logAttrs := append(append([]any{}, attrs...), "attempt", attempt+1, "backoff", backoff)
		shared.Logger.Warn("rate limited, retrying", logAttrs...)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return zero, err
		}
	}

	return zero, fmt.Errorf("unreachable")
}

func isRateLimited(err error) bool {
	var use *web.UnexpectedStatusError
	return errors.As(err, &use) && use.StatusCode == http.StatusTooManyRequests
}

func retryBackoff(delay time.Duration, attempt int) time.Duration {
	if delay <= 0 {
		delay = time.Second
	}
	return time.Duration(attempt) * delay
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
