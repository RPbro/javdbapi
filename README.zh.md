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

快速开始：

```bash
javdbapi search --keyword VR --output console
javdbapi actor --id neRNX --filter cnsub,download --stale-after 48h
javdbapi video --path /v/ZNdEbV --output console
```

### 共享参数

| 参数            | 类型     | 默认值              | 说明                                                         |
| --------------- | -------- | ------------------- | ------------------------------------------------------------ |
| `--output`      | string   | `file`              | 输出模式：`file`、`console`、`both`                          |
| `--output-dir`  | string   | `./output`          | 输出文件目录                                                 |
| `--stale-after` | duration | `24h`               | 缓存新鲜时跳过抓取；对普通缓存时间戳，`0s` 会绕过常规新鲜度检查 |
| `--timeout`     | duration | `30s`               | HTTP 请求超时                                                |
| `--delay`       | duration | `1s`                | 请求间隔                                                     |
| `--proxy-url`   | string   | —                   | HTTP/SOCKS5 代理地址                                         |
| `--base-url`    | string   | `https://javdb.com` | 覆盖基础地址                                                 |
| `--user-agent`  | string   | —                   | 自定义 User-Agent                                            |
| `--debug`       | bool     | `false`             | 在 stderr 输出 debug 日志                                    |
| `--fail-fast`   | bool     | `false`             | 列表命令遇到第一个失败视频后立即停止                         |

### search

示例：

```bash
javdbapi search --keyword VR
javdbapi search --keyword VR --page 2 --max-pages 3 --output both
```

### home

可读值：

| 参数       | 可选值                                     | 说明                               |
| ---------- | ------------------------------------------ | ---------------------------------- |
| `--type`   | `all`、`censored`、`uncensored`、`western` | `all` 保持当前省略时的请求语义     |
| `--filter` | `all`、`download`、`cnsub`、`review`       | `all` 保持当前省略时的请求语义     |
| `--sort`   | `publish`、`magnet`                        | `publish` 保持当前省略时的请求语义 |

示例：

```bash
javdbapi home --type censored --filter all --sort publish
javdbapi home --sort magnet --output console
```

### actor

可读值：

| 参数       | 可选值                                           | 说明                           |
| ---------- | ------------------------------------------------ | ------------------------------ |
| `--filter` | `all`、`playable`、`single`、`download`、`cnsub` | 逗号分隔；兼容旧别名 `p,s,d,c` |

示例：

```bash
javdbapi actor --id neRNX --filter cnsub,download
javdbapi actor --id neRNX --filter c,d
```

### maker

可读值：

| 参数       | 可选值                                                      | 说明                           |
| ---------- | ----------------------------------------------------------- | ------------------------------ |
| `--filter` | `all`、`playable`、`single`、`download`、`cnsub`、`preview` | `all` 保持当前省略时的请求语义 |

示例：

```bash
javdbapi maker --id 7R
javdbapi maker --id 7R --filter playable --output both
```

### ranking

可读值：

| 参数       | 可选值                              | 说明  |
| ---------- | ----------------------------------- | ----- |
| `--period` | `daily`、`weekly`、`monthly`        | 必填  |
| `--type`   | `censored`、`uncensored`、`western` | 必填  |

示例：

```bash
javdbapi ranking --period weekly --type censored
javdbapi ranking --period daily --type western --stale-after 0s --output console
```

### video

规则：

- `--path` 与 `--url` 必须二选一
- 使用 `--url` 时，其 host 必须和 `--base-url` 推导出的 host 一致

示例：

```bash
javdbapi video --path /v/ZNdEbV --output console
javdbapi video --url https://javdb.com/v/ZNdEbV --base-url https://javdb.com --output both
```

### AI / 程序化使用

- 对数据命令来说，`console` / `both` 模式下 stdout 只输出 JSON。
- `help` 与 `version` 仍然会向 stdout 输出文本。
- stderr 不是 JSON，不应直接喂给 parser。
- 即使最终 exit code 是 `1`，stdout 也可能已经输出了部分有效 NDJSON。
- exit code 为 `0` 且 stdout 为空，可能表示命中新鲜缓存。
- `--stale-after 0s` 可用于绕过常规的新鲜缓存判断。
- 对 `video` 命令来说，`--path` 与 `--url` 必须二选一。
- 对 `video` 命令来说，`--url` 的 host 必须和 `--base-url` 推导出的 host 一致。
- 本轮只保证枚举校验错误使用稳定的 `invalid --<flag> "<value>": <reason>` 格式。

程序化示例：

```bash
javdbapi video --path /v/ZNdEbV --output console | jq '.video.code'
javdbapi video --url https://javdb.com/v/ZNdEbV --proxy-url http://127.0.0.1:7890 --output console | jq '.metadata.path'
javdbapi ranking --period weekly --type censored --stale-after 0s --output console | jq -c '.video.code'
```

## 测试策略

### 默认测试

```bash
go test ./...
```

默认测试是离线且可重复的，依赖本地 fixture 和 `httptest`，不需要外网。
