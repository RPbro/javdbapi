package clioutput

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
)

type Document struct {
	Metadata Metadata       `json:"metadata"`
	Video    javdbapi.Video `json:"video"`
}

type Metadata struct {
	LastUpdated time.Time `json:"last_updated"`
	Path        string    `json:"path"`
	PathKey     string    `json:"path_key"`
	Sources     []Source  `json:"sources"`
}

type Source struct {
	Command string          `json:"command"`
	Query   json.RawMessage `json:"query"`
}

func (s Source) Key() string {
	var compact bytes.Buffer
	if err := json.Compact(&compact, s.Query); err != nil {
		return s.Command + ":" + string(s.Query)
	}
	return s.Command + ":" + compact.String()
}

type searchSourceQuery struct {
	Keyword string `json:"keyword"`
	Page    int    `json:"page"`
}

type homeSourceQuery struct {
	Type   string `json:"type,omitempty"`
	Filter string `json:"filter,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Page   int    `json:"page"`
}

type makerSourceQuery struct {
	ID     string `json:"id"`
	Filter string `json:"filter,omitempty"`
	Page   int    `json:"page"`
}

type actorSourceQuery struct {
	ID     string   `json:"id"`
	Filter []string `json:"filter,omitempty"`
	Page   int      `json:"page"`
}

type rankingSourceQuery struct {
	Period string `json:"period"`
	Type   string `json:"type"`
	Page   int    `json:"page"`
}

type videoSourceQuery struct {
	Path string `json:"path"`
}

func NewHomeSource(typ string, filter string, sort string, page int) Source {
	return mustMarshalSource("home", homeSourceQuery{
		Type:   typ,
		Filter: filter,
		Sort:   sort,
		Page:   page,
	})
}

func NewMakerSource(id string, filter string, page int) Source {
	return mustMarshalSource("maker", makerSourceQuery{
		ID:     id,
		Filter: filter,
		Page:   page,
	})
}

func NewSearchSource(keyword string, page int) Source {
	return mustMarshalSource("search", searchSourceQuery{
		Keyword: keyword,
		Page:    page,
	})
}

func NewActorSource(id string, filters []string, page int) Source {
	cleaned := make([]string, 0, len(filters))
	seen := make(map[string]struct{}, len(filters))
	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}
		if _, ok := seen[filter]; ok {
			continue
		}
		seen[filter] = struct{}{}
		cleaned = append(cleaned, filter)
	}
	slices.Sort(cleaned)
	return mustMarshalSource("actor", actorSourceQuery{
		ID:     id,
		Filter: cleaned,
		Page:   page,
	})
}

func NewRankingSource(period string, typ string, page int) Source {
	return mustMarshalSource("ranking", rankingSourceQuery{
		Period: period,
		Type:   typ,
		Page:   page,
	})
}

func NewVideoSource(path string) Source {
	return mustMarshalSource("video", videoSourceQuery{Path: path})
}

func mustMarshalSource(command string, query any) Source {
	raw, err := json.Marshal(query)
	if err != nil {
		panic(err)
	}
	return Source{Command: command, Query: raw}
}
