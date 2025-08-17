package javdbapi

import (
	"time"
)

type Filter struct {
	pass bool

	// list
	ScoreGT       float64
	ScoreLT       float64
	ScoreCountGT  int
	ScoreCountLT  int
	PubDateBefore time.Time
	PubDateAfter  time.Time
	HasSubtitle   bool

	// details
	HasPreview  bool
	ActorsIn    []string
	ActorsNotIn []string
	TagsIn      []string
	TagsNotIn   []string
	HasPics     bool
	HasMagnets  bool

	// reviews
	HasReviews bool

	// result
	ResultRegexpMagnets string
}

func (f *Filter) checkScoreGT(in float64) {
	if f.ScoreGT > 0 && in < f.ScoreGT {
		f.pass = false
	}
}

func (f *Filter) checkScoreLT(in float64) {
	if f.ScoreLT > 0 && in > f.ScoreLT {
		f.pass = false
	}
}

func (f *Filter) checkScoreCountGT(in int) {
	if f.ScoreCountGT > 0 && in < f.ScoreCountGT {
		f.pass = false
	}
}

func (f *Filter) checkScoreCountLT(in int) {
	if f.ScoreCountLT > 0 && in > f.ScoreCountLT {
		f.pass = false
	}
}

func (f *Filter) checkPubDateBefore(in time.Time) {
	if !f.PubDateBefore.IsZero() && in.After(f.PubDateBefore) {
		f.pass = false
	}
}

func (f *Filter) checkPubDateAfter(in time.Time) {
	if !f.PubDateAfter.IsZero() && in.Before(f.PubDateAfter) {
		f.pass = false
	}
}

func (f *Filter) checkHasSubtitle(in bool) {
	if f.HasSubtitle && !in {
		f.pass = false
	}
}

func (f *Filter) checkHasPreview(in string) {
	if f.HasPreview && in == "" {
		f.pass = false
	}
}

func (f *Filter) checkActorsIn(in []string) {
	if f.ActorsIn != nil && !sliceContainsAny(in, f.ActorsIn) {
		f.pass = false
	}
}

func (f *Filter) checkActorsNotIn(in []string) {
	if f.ActorsNotIn != nil && sliceContainsAny(in, f.ActorsNotIn) {
		f.pass = false
	}
}

func (f *Filter) checkTagsIn(in []string) {
	if f.TagsIn != nil && !sliceContainsAny(in, f.TagsIn) {
		f.pass = false
	}
}

func (f *Filter) checkTagsNotIn(in []string) {
	if f.TagsNotIn != nil && sliceContainsAny(in, f.TagsNotIn) {
		f.pass = false
	}
}

func (f *Filter) checkHasPics(in []string) {
	if f.HasPics && len(in) == 0 {
		f.pass = false
	}
}

func (f *Filter) checkHasMagnets(in []string) {
	if f.HasMagnets && len(in) == 0 {
		f.pass = false
	}
}

func (f *Filter) checkHasReviews(in []string) {
	if f.HasReviews && len(in) == 0 {
		f.pass = false
	}
}

func (f *Filter) checkResultRegexpMagnets(in []string) {
	for _, v := range in {
		if f.ResultRegexpMagnets != "" && strIsMatch(v, f.ResultRegexpMagnets) {
			return
		}
	}
	f.pass = false
}
