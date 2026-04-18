package cliapp

import (
	"context"

	javdbapi "github.com/RPbro/javdbapi"
)

type Fetcher interface {
	Home(context.Context, javdbapi.HomeQuery) ([]javdbapi.Video, error)
	Search(context.Context, javdbapi.SearchQuery) ([]javdbapi.Video, error)
	Maker(context.Context, javdbapi.MakerQuery) ([]javdbapi.Video, error)
	Actor(context.Context, javdbapi.ActorQuery) ([]javdbapi.Video, error)
	Ranking(context.Context, javdbapi.RankingQuery) ([]javdbapi.Video, error)
	Video(context.Context, javdbapi.VideoQuery) (*javdbapi.Video, error)
}

type ClientFetcher struct {
	Client *javdbapi.Client
}

func (f ClientFetcher) Home(ctx context.Context, q javdbapi.HomeQuery) ([]javdbapi.Video, error) {
	return f.Client.Home(ctx, q)
}

func (f ClientFetcher) Search(ctx context.Context, q javdbapi.SearchQuery) ([]javdbapi.Video, error) {
	return f.Client.Search(ctx, q)
}

func (f ClientFetcher) Maker(ctx context.Context, q javdbapi.MakerQuery) ([]javdbapi.Video, error) {
	return f.Client.Maker(ctx, q)
}

func (f ClientFetcher) Actor(ctx context.Context, q javdbapi.ActorQuery) ([]javdbapi.Video, error) {
	return f.Client.Actor(ctx, q)
}

func (f ClientFetcher) Ranking(ctx context.Context, q javdbapi.RankingQuery) ([]javdbapi.Video, error) {
	return f.Client.Ranking(ctx, q)
}

func (f ClientFetcher) Video(ctx context.Context, q javdbapi.VideoQuery) (*javdbapi.Video, error) {
	return f.Client.Video(ctx, q)
}
