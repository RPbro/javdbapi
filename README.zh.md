# javdbapi

`javdbapi` 是一个用于查询 `javdb.com` 的 Go library，公开接口由显式 `Client`、强类型 ID 和强类型查询对象组成。视频**摘要**（列表页）、**详情**（演员、标签、截图、磁力链接）与**评论**是三个相互独立的请求。

## 环境要求

- Go `1.26.5`

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

`BaseURL`、`Locale`、`UserAgent`、`HTTP.Timeout`、`HTTP.MaxResponseBytes`、`Retry`、`RateLimit` 留空（零值）时都会使用默认值。`HTTP.Client` 不能与 `HTTP.Timeout` 或 `HTTP.ProxyURL` 同时设置——二选一。

## 强类型 ID

每一种资源 ID（`VideoID`、`ActorID`、`MakerID`、`DirectorID`、`SeriesID`）都是独立的字符串类型，只能通过对应的 `Parse*` 函数构造。SDK 从不接受任意的绝对 URL 作为输入——只接受经过校验的站内 ID：

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

要点：

- `Home`、`Search`、`MakerVideos`、`ActorVideos`、`Ranking` 都返回 `Page[VideoSummary]`。`Page` 从不声明总页数——站点只提供 `rel="next"` 分页标记，请通过 `HasNext` 判断是否继续翻页。
- `Detail` 与 `Reviews` 是针对 `VideoID` 的两个独立请求。获取 `Detail` 不会隐式拉取 `Reviews`；`Reviews` 拉取失败也不会使已经拉取成功的 `Detail` 失效。
- `VideoDetail` 中的可选字段（`Director`、`Maker`、`Series`）缺失时为 `nil` 指针；切片字段（`Actors`、`Tags`、`Screenshots`、`Magnets`）始终是非 `nil` 的空切片——包括 JSON 序列化时，始终编码为 `[]`，而不是 `null`。
- `VideoSummary.Score` 是 `*Score`：`nil` 表示页面尚未产生评分（新作品的正常状态）；非 `nil` 且 `Value == 0` 表示站点明确给出了零分。使用前务必先判断 `nil`，两种状态不可混淆。
- 同一个 `*Client` 发出的所有请求共享同一个限流器，因此并发调用者永远不会超过配置的请求预算。

## 错误处理

哨兵错误（`ErrInvalidConfig`、`ErrInvalidQuery`、`ErrNotFound`、`ErrRateLimited`、`ErrEmptyResult`、`ErrParse`、`ErrChallenge`、`ErrAuthenticationRequired`）通过 `errors.Is` 判断。`OpError` 包装失败的操作名以及（在安全的情况下）去除了查询参数的请求 URL；`HTTPError` 报告原始的非 2xx 状态码，并通过 `errors.Is` 将 404/429 映射到 `ErrNotFound`/`ErrRateLimited`。`ErrChallenge` 表示响应被判定为访问限制（例如 Cloudflare challenge）而非目标页面本身；SDK 不会尝试破解或绕过该限制，调用方应更换出口网络、配置可用代理或稍后重试。`ErrAuthenticationRequired` 表示目标资源重定向到了站点登录页；SDK 不会继续请求登录页，也不会把登录页交给解析器。

## CLI

安装：

```bash
go install github.com/RPbro/javdbapi/cmd/javdbapi@latest
```

快速开始：

```bash
javdbapi search --keyword VR --output console
javdbapi actor --id neRNX --filter cnsub,download --stale-after 48h
javdbapi video --id ZNdEbV --output console
```

### 共享参数

| 参数            | 类型     | 默认值              | 说明                                                            |
| --------------- | -------- | ------------------- | --------------------------------------------------------------- |
| `--output`      | string   | `file`              | 输出模式：`file`、`console`、`both`                             |
| `--output-dir`  | string   | `./output`          | 输出文件目录                                                    |
| `--stale-after` | duration | `24h`               | 缓存新鲜时跳过抓取；对普通缓存时间戳，`0s` 会绕过常规新鲜度检查 |
| `--concurrency` | int      | `2`                 | 列表命令并发抓取视频的数量，取值范围 `1`-`16`                   |
| `--timeout`     | duration | `30s`               | HTTP 请求超时                                                   |
| `--proxy-url`   | string   | —                   | HTTP/SOCKS5 代理地址                                            |
| `--base-url`    | string   | `https://javdb.com` | 覆盖基础地址                                                    |
| `--user-agent`  | string   | —                   | 自定义 User-Agent                                               |
| `--rate`        | float    | `1`                 | 一次命令调用内所有 worker 共享的每秒请求数                      |
| `--burst`       | int      | `1`                 | 限流器的突发容量                                                |
| `--debug`       | bool     | `false`             | 在 stderr 输出 debug 日志；同时提升 SDK client 自身的日志级别   |
| `--fail-fast`   | bool     | `false`             | 遇到第一个硬失败立即停止，而不是持续累计失败                    |

`--concurrency` 限制的是一次列表命令内并发抓取视频的数量；`--rate`/`--burst` 限制的是共享 `Client` 整体的请求速率。只调高 `--concurrency` 而不调高 `--rate`，多半只是增加了排队等待，而不会真正提升抓取吞吐量。

### `--summary-only`

所有列表命令（`search`、`home`、`maker`、`actor`、`ranking`）都支持该参数，`video` 不支持。必须配合 `--output console` 使用；与 `--output file` 或 `--output both` 同时使用会返回稳定错误 `--summary-only requires --output console`。

- 完全跳过 `Detail` 与 `Reviews`，只请求列表页本身。
- 不触碰磁盘缓存：不读取、不写入，也不创建输出目录。
- 分页与按 `VideoID` 去重的行为与默认模式完全一致，随后按发现顺序把每个去重后的 `VideoSummary` 输出为一行 NDJSON。
- 命令汇总中的 `SummariesOutput` 字段记录实际输出的摘要数量。

```bash
javdbapi search --keyword VR --summary-only --output console
```

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

| 参数       | 可选值                              | 说明 |
| ---------- | ----------------------------------- | ---- |
| `--period` | `daily`、`weekly`、`monthly`        | 必填 |
| `--type`   | `censored`、`uncensored`、`western` | 必填 |

示例：

```bash
javdbapi ranking --period weekly --type censored
javdbapi ranking --period daily --type western --stale-after 0s --output console
```

### video

规则：

- `--id` 必填，且必须是纯粹的站内视频 ID（例如 `ZNdEbV`），而不是路径或 URL。

示例：

```bash
javdbapi video --id ZNdEbV --output console
javdbapi video --id ZNdEbV --base-url https://javdb.com --output both
```

每个视频的缓存文档会独立追踪 `Detail` 与 `Reviews` 的新鲜度。如果上一次运行已经成功持久化 `Detail`，但 `Reviews` 拉取失败，失败信息会记录在缓存文档的 `partial_errors` 字段中；下一次调用只会重新拉取过期的 `Reviews`，并复用已经是新鲜状态的 `Detail`。当 `Reviews` 请求被访问限制拦截时，`partial_errors[].kind` 会是 `challenge`；重定向到登录页时会是 `authentication_required`；其余可能值还包括 `not_found`、`rate_limited`、`empty_result`、`parse_error`、`fetch_error`。

### AI / 程序化使用

- 对数据命令来说，`console` / `both` 模式下 stdout 只输出 JSON。
- `help` 与 `version` 仍然会向 stdout 输出文本。
- stderr 不是 JSON，不应直接喂给 parser。
- 即使最终 exit code 是 `1`，stdout 也可能已经输出了部分有效 NDJSON。
- exit code 为 `0` 且 stdout 为空，可能表示命中新鲜缓存。
- `--stale-after 0s` 可用于绕过常规的新鲜缓存判断。
- 对 `video` 命令来说，`--id` 必填，且必须是纯粹的站内视频 ID。
- 本轮只保证枚举校验错误使用稳定的 `invalid --<flag> "<value>": <reason>` 格式。
- `--concurrency` 超出 `1`-`16` 范围时，会返回稳定的错误信息 `--concurrency must be between 1 and 16`。
- 一个视频即使 `Reviews` 拉取失败，只要 `Detail` 成功，默认情况下依然会正常持久化；只有 `--fail-fast` 才会把 `Reviews` 失败提升为命令级错误。

程序化示例：

```bash
javdbapi video --id ZNdEbV --output console | jq '.detail.summary.code'
javdbapi video --id ZNdEbV --proxy-url http://127.0.0.1:7890 --output console | jq '.metadata.sources'
javdbapi ranking --period weekly --type censored --stale-after 0s --output console | jq -c '.detail.summary.code'
```

## 缓存格式

缓存文档使用 `"schema_version": 2` 写入，并采用原子写入（先写临时文件再 rename），因此中途崩溃永远不会破坏已有的缓存文件。v1 缓存文件（缺少或不匹配 `schema_version`）一律视为缓存缺失——没有迁移路径。`detail_updated_at`、`reviews_updated_at`、`reviews_last_attempted_at` 各自独立追踪新鲜度；数组字段（`reviews`、`partial_errors`、`sources`，以及 `detail` 子对象自身的数组字段）始终序列化为 `[]`，而不是 `null`。

## 测试策略

### 默认测试

```bash
go test ./...
```

默认测试是离线且可重复的，依赖本地 fixture 和 `httptest`，不需要外网。

### 竞态检测

```bash
make race
```

以 `-race` 运行完整测试集，覆盖 CLI 的并发 worker pool 和共享限流器。

### 可选的真实环境集成测试

```bash
make integration
```

运行 `internal/scrape/integration_test.go` 中的 `TestLiveSearchContract`，针对真实站点发起请求。除非设置了 `JAVDB_INTEGRATION=1`，否则该测试会被跳过；它不属于默认的 `test` 目标，也不会在 CI 中运行。此处出现 `403`、`429` 或网络错误，反映的是站点访问/环境限制，而不是 fixture parser 的回归问题。
