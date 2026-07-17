# javdbapi

`javdbapi` is a Go library for querying `javdb.com` with an explicit `Client`, typed IDs, and typed query objects. It separates video **summaries** (from list pages), **detail** (actors, tags, screenshots, magnets), and **reviews** into independent requests.

## Requirements

- Go `1.26.5`

## Install

```bash
go get github.com/RPbro/javdbapi
```

## Initialize Client

```go
package main

import (
	"log"
	"time"

	"github.com/RPbro/javdbapi"
)

func main() {
	client, err := javdbapi.NewClient(javdbapi.ClientConfig{
		BaseURL:   "https://javdb.com",
		UserAgent: "",
		HTTP: javdbapi.HTTPConfig{
			Timeout:  30 * time.Second,
			ProxyURL: "",
		},
		RateLimit: javdbapi.RateLimitPolicy{
			RequestsPerSecond: 1,
			Burst:             1,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = client
}
```

`BaseURL`, `Locale`, `UserAgent`, `HTTP.Timeout`, `HTTP.MaxResponseBytes`, `Retry`, and `RateLimit` all have defaults when left zero-valued. `HTTP.Client` cannot be combined with `HTTP.Timeout` or `HTTP.ProxyURL` — supply one or the other, not both.

## Typed IDs

Every resource ID (`VideoID`, `ActorID`, `MakerID`, `DirectorID`, `SeriesID`) is a distinct string type, constructed only through its `Parse*` function. The SDK never accepts arbitrary absolute URLs as input — only validated in-site IDs:

```go
videoID, err := javdbapi.ParseVideoID("ZNdEbV")
if err != nil {
	log.Fatal(err)
}

fmt.Println(client.VideoURL(videoID)) // https://javdb.com/v/ZNdEbV?locale=zh
```

## Library API

```go
package main

import (
	"context"
	"log"

	"github.com/RPbro/javdbapi"
)

func main() {
	client, err := javdbapi.NewClient(javdbapi.ClientConfig{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	home, err := client.Home(ctx, javdbapi.HomeQuery{
		Type:   javdbapi.HomeTypeAll,
		Filter: javdbapi.HomeFilterAll,
		Sort:   javdbapi.HomeSortPublishDate,
		Page:   1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("home: %d items, has_next=%v", len(home.Items), home.HasNext)

	search, err := client.Search(ctx, javdbapi.SearchQuery{
		Keyword: "VR",
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("search: %d items", len(search.Items))

	makerID, err := javdbapi.ParseMakerID("7R")
	if err != nil {
		log.Fatal(err)
	}
	maker, err := client.MakerVideos(ctx, javdbapi.MakerVideosQuery{
		MakerID: makerID,
		Filter:  javdbapi.MakerFilterAll,
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("maker: %d items", len(maker.Items))

	actorID, err := javdbapi.ParseActorID("neRNX")
	if err != nil {
		log.Fatal(err)
	}
	actor, err := client.ActorVideos(ctx, javdbapi.ActorVideosQuery{
		ActorID: actorID,
		Filters: []javdbapi.ActorFilter{javdbapi.ActorFilterAll},
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("actor: %d items", len(actor.Items))

	ranking, err := client.Ranking(ctx, javdbapi.RankingQuery{
		Period: javdbapi.RankingPeriodWeekly,
		Type:   javdbapi.RankingTypeCensored,
		Page:   1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ranking: %d items", len(ranking.Items))

	videoID, err := javdbapi.ParseVideoID("ZNdEbV")
	if err != nil {
		log.Fatal(err)
	}

	detail, err := client.Detail(ctx, videoID)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("detail: %s, %d actors, %d magnets", detail.Summary.Code, len(detail.Actors), len(detail.Magnets))
	if detail.Summary.Score != nil {
		log.Printf("score: %.2f (%d ratings)", detail.Summary.Score.Value, detail.Summary.Score.Count)
	} else {
		log.Printf("score: not yet available")
	}

	reviews, err := client.Reviews(ctx, videoID)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("reviews: %d", len(reviews))
}
```

Key points:

- `Home`, `Search`, `MakerVideos`, `ActorVideos`, and `Ranking` all return `Page[VideoSummary]`. `Page` never claims a total page count — the site only exposes `rel="next"` pagination, so check `HasNext` to decide whether to keep paginating.
- `Detail` and `Reviews` are independent requests against `VideoID`. Fetching `Detail` never implicitly fetches `Reviews`, and a `Reviews` failure never invalidates an already-fetched `Detail`.
- Optional `VideoDetail` sections (`Director`, `Maker`, `Series`) are `nil` pointers when absent; slice sections (`Actors`, `Tags`, `Screenshots`, `Magnets`) are always non-nil, empty slices rather than `nil` — including in JSON, which encodes `[]`, never `null`.
- `VideoSummary.Score` is `*Score`: `nil` means the page has no score yet (normal for new releases), while a non-nil `Score` with `Value == 0` means the site explicitly reports a zero score. Always check for `nil` before dereferencing; the two states are never conflated.
- All requests issued through one `*Client` share the same rate limiter, so concurrent callers never exceed the configured request budget.

## Errors

Sentinel errors (`ErrInvalidConfig`, `ErrInvalidQuery`, `ErrNotFound`, `ErrRateLimited`, `ErrEmptyResult`, `ErrParse`, `ErrChallenge`, `ErrAuthenticationRequired`) are checked with `errors.Is`. `OpError` wraps the failing operation and (when safe) a query-stripped request URL; `HTTPError` reports the raw non-2xx status code and maps 404/429 onto `ErrNotFound`/`ErrRateLimited` via `errors.Is`. `ErrChallenge` means the response was classified as an access challenge (e.g. Cloudflare) rather than the requested page; the SDK never attempts to solve or bypass it — callers should switch network egress, supply a working proxy, or retry later. `ErrAuthenticationRequired` means the requested resource redirected to the site's login page; the login page is never fetched or passed to a scraper.

## CLI

Install:

```bash
go install github.com/RPbro/javdbapi/cmd/javdbapi@latest
```

Quick start:

```bash
javdbapi search --keyword VR --output console
javdbapi actor --id neRNX --filter cnsub,download --stale-after 48h
javdbapi video --id ZNdEbV --output console
```

### Shared Flags

| Flag            | Type     | Default             | Description                                                                                         |
| --------------- | -------- | ------------------- | --------------------------------------------------------------------------------------------------- |
| `--output`      | string   | `file`              | Output mode: `file`, `console`, `both`                                                              |
| `--output-dir`  | string   | `./output`          | Directory for output files                                                                          |
| `--stale-after` | duration | `24h`               | Skip fetch when cache is fresh; `0s` bypasses ordinary freshness checks for normal cache timestamps |
| `--concurrency` | int      | `2`                 | Number of videos fetched concurrently for list commands, `1`-`16`                                   |
| `--timeout`     | duration | `30s`               | HTTP request timeout                                                                                |
| `--proxy-url`   | string   | —                   | HTTP/SOCKS5 proxy URL                                                                               |
| `--base-url`    | string   | `https://javdb.com` | Override base URL                                                                                   |
| `--user-agent`  | string   | —                   | Custom User-Agent header                                                                            |
| `--rate`        | float    | `1`                 | Requests per second shared across all workers of one command invocation                             |
| `--burst`       | int      | `1`                 | Rate limiter burst size                                                                             |
| `--debug`       | bool     | `false`             | Enable debug logs on stderr; also raises the SDK client's own log level                             |
| `--fail-fast`   | bool     | `false`             | Stop on the first hard error instead of accumulating failures                                       |

`--concurrency` bounds how many videos are fetched in parallel within one list command; `--rate`/`--burst` bound how fast the shared `Client` issues requests overall. Raising `--concurrency` without raising `--rate` mostly increases queuing against the same request budget, not real fetch throughput.

### `--summary-only`

Available on every list command (`search`, `home`, `maker`, `actor`, `ranking`) but not on `video`. It requires `--output console`; combining it with `--output file` or `--output both` fails with the stable error `--summary-only requires --output console`.

- Skips `Detail` and `Reviews` entirely — only the list pages themselves are fetched.
- Never touches the on-disk cache: no read, no write, no output directory creation.
- Paginates and deduplicates by `VideoID` exactly like the default mode, then prints each deduplicated `VideoSummary` as one NDJSON line, in discovery order.
- The command summary's `SummariesOutput` field reports how many summaries were printed.

```bash
javdbapi search --keyword VR --summary-only --output console
```

### search

Examples:

```bash
javdbapi search --keyword VR
javdbapi search --keyword VR --page 2 --max-pages 3 --output both
```

### home

Canonical values:

| Flag       | Values                                     | Notes                                        |
| ---------- | ------------------------------------------ | -------------------------------------------- |
| `--type`   | `all`, `censored`, `uncensored`, `western` | `all` keeps the omitted request behavior     |
| `--filter` | `all`, `download`, `cnsub`, `review`       | `all` keeps the omitted request behavior     |
| `--sort`   | `publish`, `magnet`                        | `publish` keeps the omitted request behavior |

Examples:

```bash
javdbapi home --type censored --filter all --sort publish
javdbapi home --sort magnet --output console
```

### actor

Canonical values:

| Flag       | Values                                           | Notes                                                     |
| ---------- | ------------------------------------------------ | --------------------------------------------------------- |
| `--filter` | `all`, `playable`, `single`, `download`, `cnsub` | comma-separated; legacy `p,s,d,c` aliases remain accepted |

Examples:

```bash
javdbapi actor --id neRNX --filter cnsub,download
javdbapi actor --id neRNX --filter c,d
```

### maker

Canonical values:

| Flag       | Values                                                      | Notes                                    |
| ---------- | ----------------------------------------------------------- | ---------------------------------------- |
| `--filter` | `all`, `playable`, `single`, `download`, `cnsub`, `preview` | `all` keeps the omitted request behavior |

Examples:

```bash
javdbapi maker --id 7R
javdbapi maker --id 7R --filter playable --output both
```

### ranking

Canonical values:

| Flag       | Values                              | Notes    |
| ---------- | ----------------------------------- | -------- |
| `--period` | `daily`, `weekly`, `monthly`        | required |
| `--type`   | `censored`, `uncensored`, `western` | required |

Examples:

```bash
javdbapi ranking --period weekly --type censored
javdbapi ranking --period daily --type western --stale-after 0s --output console
```

### video

Rules:

- `--id` is required and must be a bare in-site video ID (e.g. `ZNdEbV`), not a path or URL.

Examples:

```bash
javdbapi video --id ZNdEbV --output console
javdbapi video --id ZNdEbV --base-url https://javdb.com --output both
```

A video's cached document tracks `Detail` and `Reviews` freshness independently. If a prior run persisted `Detail` but the `Reviews` fetch failed, the failure is recorded under `partial_errors` in the cache document and the next invocation retries only the stale `Reviews`, reusing the already-fresh `Detail`. `partial_errors[].kind` includes `challenge` when the `Reviews` request was blocked by an access challenge and `authentication_required` when it redirected to the login page, alongside `not_found`, `rate_limited`, `empty_result`, `parse_error`, and `fetch_error`.

### AI / Programmatic Usage

- For data commands, stdout is JSON-only in `console` and `both`.
- `help` and `version` write text to stdout.
- stderr is not JSON and should never be sent to a JSON parser.
- Exit code `1` may still arrive after partial valid NDJSON has already been written.
- Empty stdout with exit code `0` can mean a fresh cache hit.
- `--stale-after 0s` can be used when the caller wants to bypass ordinary fresh-cache checks.
- For `video`, `--id` is required and must be a bare in-site video ID.
- Enum validation errors use the stable `invalid --<flag> "<value>": <reason>` format in this iteration.
- `--concurrency` out of the `1`-`16` range fails with the stable message `--concurrency must be between 1 and 16`.
- A video whose `Detail` succeeded but whose `Reviews` fetch failed still persists successfully by default; only `--fail-fast` turns a `Reviews` failure into a command-level error.

Programmatic examples:

```bash
javdbapi video --id ZNdEbV --output console | jq '.detail.summary.code'
javdbapi video --id ZNdEbV --proxy-url http://127.0.0.1:7890 --output console | jq '.metadata.sources'
javdbapi ranking --period weekly --type censored --stale-after 0s --output console | jq -c '.detail.summary.code'
```

## Cache Schema

Cache documents are written with `"schema_version": 2` and use atomic writes (write to a temp file, then rename) so a crash mid-write never corrupts an existing cache file. v1 cache files (missing or mismatched `schema_version`) are always treated as a cache miss — there is no migration path. `detail_updated_at`, `reviews_updated_at`, and `reviews_last_attempted_at` track freshness independently, and array fields (`reviews`, `partial_errors`, `sources`, and the `detail` sub-object's own array fields) always serialize as `[]`, never `null`.

## Test Strategy

### Default tests

```bash
go test ./...
```

Default tests are offline and deterministic, using local fixtures and `httptest`. They do not require external network access.

### Race detector

```bash
make race
```

Runs the full suite with `-race`, covering the CLI's bounded concurrent worker pool and the shared rate limiter.

### Opt-in live integration test

```bash
make integration
```

Runs `internal/scrape/integration_test.go`'s `TestLiveSearchContract` against the real site. It is skipped unless `JAVDB_INTEGRATION=1` is set, is never part of the default `test` target, and is not run in CI. A `403`, `429`, or network error here reflects a site-access/environment limitation, not a fixture parser regression.
