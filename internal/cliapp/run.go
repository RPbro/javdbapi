package cliapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
	"github.com/RPbro/javdbapi/internal/clioutput"
)

func validateSharedOptions(shared SharedOptions) error {
	switch shared.OutputMode {
	case OutputConsole, OutputFile, OutputBoth:
		return nil
	default:
		return fmt.Errorf("invalid --output %q: must be one of console, file, both", shared.OutputMode)
	}
}

// RunListCommand paginates a list command, deduplicates discovered video IDs,
// and dispatches them through a bounded worker pool for Detail/Reviews
// fetching. Output is flushed in original discovery order regardless of
// which worker finishes first. In SummaryOnly mode, discovery output is the
// only work performed: no Detail/Reviews request or cache access ever happens.
func RunListCommand(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req ListRequest) (Summary, error) {
	if err := validateSharedOptions(req.Shared); err != nil {
		return Summary{}, err
	}
	if req.SummaryOnly && req.Shared.OutputMode != OutputConsole {
		return Summary{}, fmt.Errorf("--summary-only requires --output console")
	}
	if req.Page < 1 {
		return Summary{}, fmt.Errorf("invalid --page %d: must be >= 1", req.Page)
	}
	if req.MaxPages < 1 {
		return Summary{}, fmt.Errorf("invalid --max-pages %d: must be >= 1", req.MaxPages)
	}
	if req.Shared.Concurrency < 1 {
		return Summary{}, fmt.Errorf("invalid concurrency %d: must be >= 1", req.Shared.Concurrency)
	}

	summary := Summary{}
	refs := make([]VideoRef, 0)
	seen := make(map[javdbapi.VideoID]struct{})

	for page := req.Page; page < req.Page+req.MaxPages; page++ {
		listPage, err := fetchListPage(ctx, fetcher, req, page)
		if err != nil {
			if errors.Is(err, javdbapi.ErrEmptyResult) {
				break
			}
			return summary, err
		}

		summary.PagesScanned++
		summary.Candidates += len(listPage.Items)

		for _, item := range listPage.Items {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			refs = append(refs, VideoRef{ID: item.ID, Title: item.Title, Code: item.Code, Page: page, Summary: item})
		}

		if !listPage.HasNext {
			break
		}
	}

	if req.SummaryOnly {
		summary.Deduplicated = len(refs)
		if err := writeSummaryOnlyOutput(req.Shared.Stdout, refs, &summary); err != nil {
			return summary, err
		}
		return summary, nil
	}

	return processRefs(ctx, fetcher, store, req, refs, summary)
}

// writeSummaryOnlyOutput encodes each ref's VideoSummary as one NDJSON line
// in discovery order. It never calls Detail, Reviews, or the cache store.
func writeSummaryOnlyOutput(w io.Writer, refs []VideoRef, summary *Summary) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, ref := range refs {
		if err := enc.Encode(ref.Summary); err != nil {
			return err
		}
		summary.SummariesOutput++
	}
	return nil
}

func fetchListPage(ctx context.Context, fetcher Fetcher, req ListRequest, page int) (javdbapi.Page[javdbapi.VideoSummary], error) {
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
		return fetcher.MakerVideos(ctx, query)
	case CommandActor:
		query := *req.Actor
		query.Page = page
		return fetcher.ActorVideos(ctx, query)
	case CommandRanking:
		query := *req.Ranking
		query.Page = page
		return fetcher.Ranking(ctx, query)
	default:
		return javdbapi.Page[javdbapi.VideoSummary]{}, fmt.Errorf("unsupported command %q", req.Command)
	}
}

func sourceForList(req ListRequest, ref VideoRef) (clioutput.Source, error) {
	switch req.Command {
	case CommandSearch:
		return clioutput.NewSearchSource(req.Search.Keyword, ref.Page), nil
	case CommandHome:
		return clioutput.NewHomeSource(string(req.Home.Type), string(req.Home.Filter), string(req.Home.Sort), ref.Page), nil
	case CommandMaker:
		return clioutput.NewMakerSource(string(req.Maker.MakerID), string(req.Maker.Filter), ref.Page), nil
	case CommandActor:
		filters := make([]string, 0, len(req.Actor.Filters))
		for _, filter := range req.Actor.Filters {
			filters = append(filters, string(filter))
		}
		return clioutput.NewActorSource(string(req.Actor.ActorID), filters, ref.Page), nil
	case CommandRanking:
		return clioutput.NewRankingSource(string(req.Ranking.Period), string(req.Ranking.Type), ref.Page), nil
	default:
		return clioutput.Source{}, fmt.Errorf("unsupported command %q", req.Command)
	}
}

type detailJob struct {
	Index int
	Ref   VideoRef
}

type detailResult struct {
	Index    int
	Document clioutput.Document
	Persist  bool
	Summary  Summary
	Err      error
}

// processRefs runs refs through req.Shared.Concurrency workers. Results are
// buffered by index and flushed to persistDocument strictly in discovery
// order, so console/file output never depends on which worker finishes
// first. In fail-fast mode, the first hard error cancels the shared context
// and callers wait for all in-flight workers before returning.
func processRefs(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req ListRequest, refs []VideoRef, summary Summary) (Summary, error) {
	summary.Deduplicated = len(refs)
	if len(refs) == 0 {
		return summary, nil
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	concurrency := req.Shared.Concurrency
	if concurrency > len(refs) {
		concurrency = len(refs)
	}

	jobs := make(chan detailJob)
	results := make(chan detailResult, len(refs))

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				source, err := sourceForList(req, job.Ref)
				if err != nil {
					results <- detailResult{Index: job.Index, Summary: Summary{Failed: 1}, Err: err}
					continue
				}
				doc, persist, s, err := fetchDetail(workerCtx, fetcher, store, req.Shared, job.Ref, []clioutput.Source{source})
				results <- detailResult{Index: job.Index, Document: doc, Persist: persist, Summary: s, Err: err}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, ref := range refs {
			select {
			case jobs <- detailJob{Index: i, Ref: ref}:
			case <-workerCtx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	pending := make(map[int]detailResult, len(refs))
	next := 0
	var firstErr error

	for result := range results {
		pending[result.Index] = result

		for {
			res, ok := pending[next]
			if !ok {
				break
			}
			delete(pending, next)
			ref := refs[res.Index]
			next++

			summary.Fetched += res.Summary.Fetched
			summary.SkippedFresh += res.Summary.SkippedFresh
			summary.Failed += res.Summary.Failed
			summary.PartialFailed += res.Summary.PartialFailed

			if res.Persist {
				if err := persistDocument(store, req.Shared, res.Document); err != nil {
					summary.Failed++
					if firstErr == nil {
						firstErr = err
						cancel()
					}
					continue
				}
			}

			if res.Err != nil {
				req.Shared.Logger.Warn("video failed", "id", ref.ID, "error", res.Err)
				if req.Shared.FailFast && firstErr == nil {
					firstErr = res.Err
					cancel()
				}
			}
		}
	}

	if firstErr != nil {
		return summary, firstErr
	}
	if summary.Failed > 0 {
		return summary, fmt.Errorf("list command completed with %d failed videos", summary.Failed)
	}
	return summary, nil
}

// RunVideoCommand fetches a single video by its already-validated ID.
func RunVideoCommand(ctx context.Context, fetcher Fetcher, store *clioutput.Store, req VideoRequest) (Summary, error) {
	if err := validateSharedOptions(req.Shared); err != nil {
		return Summary{}, err
	}
	if req.ID == "" {
		return Summary{}, fmt.Errorf("video id is required")
	}

	ref := VideoRef{ID: req.ID}
	doc, persist, summary, fetchErr := fetchDetail(ctx, fetcher, store, req.Shared, ref, []clioutput.Source{clioutput.NewVideoSource(string(req.ID))})

	if persist {
		if err := persistDocument(store, req.Shared, doc); err != nil {
			summary.Failed++
			return summary, err
		}
	}
	if fetchErr != nil {
		return summary, fetchErr
	}
	return summary, nil
}

// fetchDetail loads the cache state for ref.ID, refreshes whichever of
// Detail/Reviews is stale, and returns the resulting Document. A Reviews
// failure never discards a successful Detail: persist is true and the
// failure is recorded as a PartialError, with Failed left untouched so the
// video is not counted as a full failure.
func fetchDetail(ctx context.Context, fetcher Fetcher, store *clioutput.Store, shared SharedOptions, ref VideoRef, sources []clioutput.Source) (doc clioutput.Document, persist bool, summary Summary, err error) {
	state, loadErr := loadCacheForMode(store, shared, ref.ID)
	if loadErr != nil {
		summary.Failed++
		return clioutput.Document{}, false, summary, loadErr
	}

	if state.DetailFresh && state.ReviewsFresh {
		summary.SkippedFresh++
		return clioutput.Document{}, false, summary, nil
	}

	if state.Document != nil {
		doc = *state.Document
	}
	doc.SchemaVersion = clioutput.SchemaVersion

	now := time.Now().UTC()

	if !state.DetailFresh {
		shared.Logger.Info("fetching video", "id", ref.ID)
		detail, detailErr := fetcher.Detail(ctx, ref.ID)
		if detailErr != nil {
			summary.Failed++
			return clioutput.Document{}, false, summary, detailErr
		}
		doc.Detail = *detail
		doc.Metadata.DetailUpdatedAt = now
	}

	doc.Metadata.Sources = clioutput.MergeSources(doc.Metadata.Sources, sources)
	doc.Metadata.LastUpdated = now

	if state.ReviewsFresh {
		summary.Fetched++
		return doc, true, summary, nil
	}

	attempted := now
	doc.Metadata.ReviewsLastAttemptedAt = &attempted

	reviews, reviewErr := fetcher.Reviews(ctx, ref.ID)
	if reviewErr != nil {
		kind := reviewErrorKind(reviewErr)
		doc.PartialErrors = []clioutput.PartialError{{
			Component: "reviews",
			Kind:      kind,
			Message:   reviewErrorMessage(kind),
		}}
		summary.PartialFailed++
		if shared.FailFast {
			return doc, true, summary, reviewErr
		}
		return doc, true, summary, nil
	}

	doc.Reviews = reviews
	reviewsUpdatedAt := now
	doc.Metadata.ReviewsUpdatedAt = &reviewsUpdatedAt
	doc.PartialErrors = nil
	summary.Fetched++
	return doc, true, summary, nil
}

func reviewErrorKind(err error) string {
	switch {
	case errors.Is(err, javdbapi.ErrChallenge):
		return "challenge"
	case errors.Is(err, javdbapi.ErrAuthenticationRequired):
		return "authentication_required"
	case errors.Is(err, javdbapi.ErrNotFound):
		return "not_found"
	case errors.Is(err, javdbapi.ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, javdbapi.ErrEmptyResult):
		return "empty_result"
	case errors.Is(err, javdbapi.ErrParse):
		return "parse_error"
	default:
		return "fetch_error"
	}
}

// reviewErrorMessage returns a fixed, stable summary for kind rather than the
// underlying error's own text, so a PartialError can never surface the
// request URL, search keyword, or response body the wrapped error may carry.
func reviewErrorMessage(kind string) string {
	switch kind {
	case "challenge":
		return "reviews request was challenged"
	case "authentication_required":
		return "reviews request requires authentication"
	case "not_found":
		return "reviews page not found"
	case "rate_limited":
		return "reviews request was rate limited"
	case "empty_result":
		return "reviews page returned no content"
	case "parse_error":
		return "failed to parse reviews page"
	default:
		return "failed to fetch reviews"
	}
}

// loadCacheForMode skips creating the output directory just to check
// freshness in pure console mode, so a read-only preview never has the
// side effect of creating a cache directory on disk.
func loadCacheForMode(store *clioutput.Store, shared SharedOptions, id javdbapi.VideoID) (clioutput.CacheState, error) {
	if shared.OutputMode == OutputConsole {
		if _, err := os.Stat(shared.OutputDir); errors.Is(err, os.ErrNotExist) {
			filePath, pathErr := store.FilePath(string(id))
			if pathErr != nil {
				return clioutput.CacheState{}, pathErr
			}
			return clioutput.CacheState{FilePath: filePath}, nil
		}
	}
	return store.Load(string(id), shared.StaleAfter)
}

func persistDocument(store *clioutput.Store, shared SharedOptions, doc clioutput.Document) error {
	if shared.OutputMode == OutputFile || shared.OutputMode == OutputBoth {
		if err := store.WriteFile(doc); err != nil {
			return err
		}
	}
	if shared.OutputMode == OutputConsole || shared.OutputMode == OutputBoth {
		if err := store.WriteJSON(shared.Stdout, doc); err != nil {
			return err
		}
	}
	return nil
}
