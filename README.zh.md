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

## 测试策略

### 默认测试

```bash
go test ./...
```

默认测试是离线且可重复的，依赖本地 fixture 和 `httptest`，不需要外网。

### 手动真实验证

手动验证会对线上 endpoint 发起真实请求，并且有意与默认测试隔离。

执行全部 endpoint：

```bash
go run ./cmd/manualtest
```

只执行部分 endpoint：

```bash
go run ./cmd/manualtest -only search,video
```

也可以通过环境变量选择：

```bash
JAVDB_MANUAL_ONLY=search,video go run ./cmd/manualtest
```

## 手动验证环境变量

Client 配置：

- `JAVDB_BASE_URL`：可选，默认 `https://javdb.com`
- `JAVDB_PROXY_URL`：可选，代理地址
- `JAVDB_TIMEOUT`：可选，`time.Duration` 格式（例如 `30s`），默认 `30s`
- `JAVDB_USER_AGENT`：可选，自定义 User-Agent

手动验证选择器：

- `JAVDB_MANUAL_ONLY`：可选，逗号分隔，可选值为 `home,search,maker,actor,ranking,video`

真实验证样例输入（可选）：

- `JAVDB_SAMPLE_KEYWORD`（默认：`VR`）
- `JAVDB_SAMPLE_MAKER_ID`（默认：`7R`）
- `JAVDB_SAMPLE_ACTOR_ID`（默认：`neRNX`）
- `JAVDB_SAMPLE_VIDEO_PATH`（默认：`/v/ZNdEbV`）

示例：

```bash
JAVDB_BASE_URL=https://javdb.com \
JAVDB_PROXY_URL=http://127.0.0.1:7890 \
JAVDB_TIMEOUT=30s \
JAVDB_USER_AGENT="Mozilla/5.0" \
JAVDB_SAMPLE_KEYWORD=VR \
JAVDB_SAMPLE_MAKER_ID=7R \
JAVDB_SAMPLE_ACTOR_ID=neRNX \
JAVDB_SAMPLE_VIDEO_PATH=/v/ZNdEbV \
go run ./cmd/manualtest -only search,video
```
