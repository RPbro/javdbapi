# javdbapi

`javdbapi` 是一个用于查询 `javdb.com` 的 Go library，公开接口由显式 `Client` 和强类型查询对象组成。

## 环境要求

- Go `1.26.2`

## 安装

```bash
go get github.com/RPbro/javdbapi
```

## 初始化 Client

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

`BaseURL`、`Timeout`、`UserAgent` 为空时会使用默认值。

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

`VideoQuery` 支持 `Path`（相对路径）或 `URL`（完整链接）两种方式。

列表接口（`Home`、`Search`、`Maker`、`Actor`、`Ranking`）返回的是列表页级别的 `Video` 摘要数据。如果你需要 `PreviewURL`、`Actors`、`Tags`、`Screenshots`、`Magnets`、`Reviews` 这类完整详情字段，请再调用 `Video`。

## CLI

安装：

```bash
go install github.com/RPbro/javdbapi/cmd/javdbapi@latest
```

本地运行：

```bash
go run ./cmd/javdbapi --help
```

示例：

```bash
go run ./cmd/javdbapi search --keyword VR --page 1 --max-pages 2
go run ./cmd/javdbapi maker --id 7R --filter playable --output both
go run ./cmd/javdbapi actor --id neRNX --filter c,d --stale-after 48h
go run ./cmd/javdbapi ranking --period weekly --type censored
go run ./cmd/javdbapi video --path /v/ZNdEbV --output console
go run ./cmd/javdbapi video --url https://javdb.com/v/ZNdEbV --output file
```

输出文件默认写入 `./output` 目录，使用 `metadata + video` 结构：

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

说明：

- `--stale-after` 使用 Go `time.Duration` 格式，例如 `30m`、`90m`、`1h30m`。
- `console` 模式命中新鲜缓存时不会向 stdout 回显 JSON，跳过原因写入 stderr。

## 测试策略

### 默认测试

```bash
go test ./...
```

默认测试是离线且可重复的，依赖本地 fixture 和 `httptest`，不需要外网。
