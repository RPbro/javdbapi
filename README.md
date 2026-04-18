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

Or run locally:

```bash
go run ./cmd/javdbapi --help
```

Examples:

```bash
go run ./cmd/javdbapi search --keyword VR --page 1 --max-pages 2
go run ./cmd/javdbapi maker --id 7R --filter playable --output both
go run ./cmd/javdbapi actor --id neRNX --filter c,d --stale-after 48h
go run ./cmd/javdbapi ranking --period weekly --type censored
go run ./cmd/javdbapi video --path /v/ZNdEbV --output console
go run ./cmd/javdbapi video --url https://javdb.com/v/ZNdEbV --output file
```

Output files are written to `./output` by default and use the `metadata + video` structure:

```json
{
  "metadata": {
    "last_updated": "2026-04-18T10:20:30Z",
    "path": "/v/ZNdEbV",
    "path_key": "ZNdEbV",
    "sources": [
      {
        "command": "video",
        "query": {
          "path": "/v/ZNdEbV"
        }
      }
    ]
  },
  "video": {
    "id": "/v/ZNdEbV"
  }
}
```

Notes:

- `--stale-after` uses Go `time.Duration` syntax such as `30m`, `90m`, or `1h30m`.
- `console` mode still checks `--output-dir` for fresh cache entries; when a fresh file is found, stdout stays empty and the skip reason is logged to stderr.

## Test Strategy

### Default tests

```bash
go test ./...
```

Default tests are offline and deterministic, using local fixtures and `httptest`. They do not require external network access.
