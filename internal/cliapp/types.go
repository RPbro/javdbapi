package cliapp

import (
	"io"
	"log/slog"
	"time"

	javdbapi "github.com/RPbro/javdbapi"
)

type CommandName string

const (
	CommandHome    CommandName = "home"
	CommandSearch  CommandName = "search"
	CommandMaker   CommandName = "maker"
	CommandActor   CommandName = "actor"
	CommandRanking CommandName = "ranking"
)

type OutputMode string

const (
	OutputConsole OutputMode = "console"
	OutputFile    OutputMode = "file"
	OutputBoth    OutputMode = "both"
)

type SharedOptions struct {
	OutputMode OutputMode
	OutputDir  string
	StaleAfter time.Duration
	Timeout    time.Duration
	Delay      time.Duration
	BaseURL    string
	ProxyURL   string
	UserAgent  string
	Debug      bool
	Stdout     io.Writer
	Logger     *slog.Logger
	FailFast   bool
}

type ListRequest struct {
	Shared   SharedOptions
	Command  CommandName
	Page     int
	MaxPages int
	Home     *javdbapi.HomeQuery
	Search   *javdbapi.SearchQuery
	Maker    *javdbapi.MakerQuery
	Actor    *javdbapi.ActorQuery
	Ranking  *javdbapi.RankingQuery
}

type VideoRequest struct {
	Shared SharedOptions
	Path   string
	URL    string
}

type VideoRef struct {
	Path  string
	Title string
	Code  string
	Page  int
}

type Summary struct {
	PagesScanned int
	Candidates   int
	Deduplicated int
	Fetched      int
	SkippedFresh int
	Failed       int
}
