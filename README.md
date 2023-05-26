# javdbapi

## Installation

```shell
go get -u github.com/RPbro/javdbapi
```

## Getting started

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RPbro/javdbapi"
)

func main() {
	client := javdbapi.NewClient(
		// optional
		javdbapi.WithDomain("https://javdb008.com"),
		// optional
		javdbapi.WithUserAgent("Mozilla/5.0 (Macintosh; ..."),
		// optional
		javdbapi.WithProxy("http://127.0.0.1:7890"),
		// optional
		javdbapi.WithTimeout(time.Second*30),
	)

	// optional: use other http.client
	c := &http.Client{}
	client.SetClient(c)

	// optional: filter
	filter := Filter{
		ScoreGT:        0,
		ScoreLT:        0,
		ScoreCountGT:   0,
		ScoreCountLT:   0,
		PubDateBefore:  time.Time{},
		PubDateAfter:   time.Time{},
		HasZH:          false,
		HasPreview:     false,
		ActressesIn:    nil,
		ActressesNotIn: nil,
		TagsIn:         nil,
		TagsNotIn:      nil,
		HasPics:        false,
		HasMagnets:     false,
		HasReviews:     false,
	}

	results, err := client.GetRaw().
		WithDetails().
		WithReviews().
		SetRaw("https://javdb.com/tags?c10=1,2,3,5").
		SetPage(1).
		SetLimit(10).
		SetFilter(filter).
		Get()
	if err != nil {
		panic(err)
	}

	for _, v := range results {
		fmt.Println(v)
	}
}
```

```go
	// first
	_, err = client.GetFirst().
		WithReviews().
		SetRaw("https://javdb008.com/v/5EOxMY").
		First()
	if err != nil {
		panic(err)
	}
```

```go
	// homepage
	_, err = client.GetHomes().
		SetCategoryCensored().
		SetFilterByCanDownload().
		SetSortByMagnetUpdate().
		Get()
	if err != nil {
		panic(err)
	}
```

```go
	// actors
	_, err = client.GetActors().
		SetActor("M4Q7"). // M4Q7 is 明里つむぎ, see https://javdb.com/actors
		SetFilterHasZH().
		SetFilterPlayable().
		Get()
	if err != nil {
		panic(err)
	}
```

```go
	// makers
	_, err = client.GetMakers().
		SetMaker("7R"). // 7R is S1 NO.1 STYLE, see https://javdb.com/makers
		SetFilterSingle().
		SetFilterHasPreview().
		Get()
	if err != nil {
		panic(err)
	}
```

```go
	// rankings
	_, err = client.GetRankings().
		SetCategoryCensored().
		SetTimeMonthly().
		Get()
	if err != nil {
		panic(err)
	}
```

```go
	// search
	_, err = client.GetSearch().
		SetQuery("PRED-483").
		Get()
	if err != nil {
		panic(err)
	}
```

```go
type JavDB struct {
        req *request
        // basic
        Path       string    `json:"path"`
        Code       string    `json:"code"`
        Title      string    `json:"title"`
        Cover      string    `json:"cover"`
        Score      float64   `json:"score"`
        ScoreCount int       `json:"score_count"`
        PubDate    time.Time `json:"pub_date"`
        HasZH      bool      `json:"has_zh"`
        // WithDetails()
        Preview   string   `json:"preview"`
        Actresses []string `json:"actresses"`
        Tags      []string `json:"tags"`
        Pics      []string `json:"pics"`
        Magnets   []string `json:"magnets"`
        // WithReviews()
        Reviews []string `json:"reviews"`
}
```
