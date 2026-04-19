# javdbapi

`javdbapi` is a Go library for querying `javdb.com` with an explicit `Client` plus typed query objects.

## Requirements

- Go `1.26.2`

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
	client, err := javdbapi.NewClient(javdbapi.Config{
		BaseURL:   "https://javdb.com",
		Timeout:   30 * time.Second,
		ProxyURL:  "",
		UserAgent: "",
		Debug:     false,
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = client
}
```

`BaseURL`, `Timeout`, and `UserAgent` have defaults when empty.

## Library API

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/RPbro/javdbapi"
)

func main() {
	client, err := javdbapi.NewClient(javdbapi.Config{Timeout: 30 * time.Second})
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
	log.Printf("home: %d", len(home))

	search, err := client.Search(ctx, javdbapi.SearchQuery{
		Keyword: "VR",
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("search: %d", len(search))

	maker, err := client.Maker(ctx, javdbapi.MakerQuery{
		MakerID: "7R",
		Filter:  javdbapi.MakerFilterAll,
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("maker: %d", len(maker))

	actor, err := client.Actor(ctx, javdbapi.ActorQuery{
		ActorID: "neRNX",
		Filters: []javdbapi.ActorFilter{javdbapi.ActorFilterAll},
		Page:    1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("actor: %d", len(actor))

	ranking, err := client.Ranking(ctx, javdbapi.RankingQuery{
		Period: javdbapi.RankingPeriodWeekly,
		Type:   javdbapi.RankingTypeCensored,
		Page:   1,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ranking: %d", len(ranking))

	video, err := client.Video(ctx, javdbapi.VideoQuery{
		Path: "/v/ZNdEbV",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("video: %s", video.Code)
}
```

`VideoQuery` supports either `Path` (relative path) or `URL` (full video URL).

List endpoints (`Home`, `Search`, `Maker`, `Actor`, `Ranking`) return summary-level `Video` items from list pages. Use `Video` when you need full detail fields such as `PreviewURL`, `Actors`, `Tags`, `Screenshots`, `Magnets`, and `Reviews`.

## CLI

Install:

```bash
go install github.com/RPbro/javdbapi/cmd/javdbapi@latest
```

Quick start:

```bash
javdbapi search --keyword VR --output console
javdbapi actor --id neRNX --filter cnsub,download --stale-after 48h
javdbapi video --path /v/ZNdEbV --output console
```

### Shared Flags

| Flag            | Type     | Default             | Description                                                                                         |
| --------------- | -------- | ------------------- | --------------------------------------------------------------------------------------------------- |
| `--output`      | string   | `file`              | Output mode: `file`, `console`, `both`                                                              |
| `--output-dir`  | string   | `./output`          | Directory for output files                                                                          |
| `--stale-after` | duration | `24h`               | Skip fetch when cache is fresh; `0s` bypasses ordinary freshness checks for normal cache timestamps |
| `--timeout`     | duration | `30s`               | HTTP request timeout                                                                                |
| `--delay`       | duration | `1s`                | Delay between requests                                                                              |
| `--proxy-url`   | string   | —                   | HTTP/SOCKS5 proxy URL                                                                               |
| `--base-url`    | string   | `https://javdb.com` | Override base URL                                                                                   |
| `--user-agent`  | string   | —                   | Custom User-Agent header                                                                            |
| `--debug`       | bool     | `false`             | Enable debug logs on stderr                                                                         |
| `--fail-fast`   | bool     | `false`             | Stop list processing after the first failing video                                                  |

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

- exactly one of `--path` or `--url` is required
- when `--url` is used, its host must match the host implied by `--base-url`

Examples:

```bash
javdbapi video --path /v/ZNdEbV --output console
javdbapi video --url https://javdb.com/v/ZNdEbV --base-url https://javdb.com --output both
```

### AI / Programmatic Usage

- For data commands, stdout is JSON-only in `console` and `both`.
- `help` and `version` write text to stdout.
- stderr is not JSON and should never be sent to a JSON parser.
- Exit code `1` may still arrive after partial valid NDJSON has already been written.
- Empty stdout with exit code `0` can mean a fresh cache hit.
- `--stale-after 0s` can be used when the caller wants to bypass ordinary fresh-cache checks.
- For `video`, exactly one of `--path` or `--url` is required.
- For `video`, the host in `--url` must match the host implied by `--base-url`.
- Enum validation errors use the stable `invalid --<flag> "<value>": <reason>` format in this iteration.

Programmatic examples:

```bash
javdbapi video --path /v/ZNdEbV --output console | jq '.video.code'
javdbapi video --url https://javdb.com/v/ZNdEbV --proxy-url http://127.0.0.1:7890 --output console | jq '.metadata.path'
javdbapi ranking --period weekly --type censored --stale-after 0s --output console | jq -c '.video.code'
```

## Test Strategy

### Default tests

```bash
go test ./...
```

Default tests are offline and deterministic, using local fixtures and `httptest`. They do not require external network access.
