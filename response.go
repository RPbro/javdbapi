package javdbapi

import (
	"time"
)

type JavDB struct {
	req     *request
	deleted bool

	Path       string    `json:"path"`
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	Cover      string    `json:"cover"`
	Score      float64   `json:"score"`
	ScoreCount int       `json:"score_count"`
	PubDate    time.Time `json:"pub_date"`
	HasZH      bool      `json:"has_zh"`

	Preview   string   `json:"preview"`
	Actresses []string `json:"actresses"`
	Tags      []string `json:"tags"`
	Pics      []string `json:"pics"`
	Magnets   []string `json:"magnets"`

	Reviews []string `json:"reviews"`
}

func (j *JavDB) loadList() ([]*JavDB, error) {
	r, err := j.req.requestList()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (j *JavDB) loadDetails() error {
	if j.deleted {
		return nil
	}
	r, err := j.req.requestDetails()
	if err != nil {
		return err
	}
	j.Preview = r.Preview
	j.Actresses = r.Actresses
	j.Tags = r.Tags
	j.Pics = r.Pics
	j.Magnets = r.Magnets

	if len(j.Path) == 0 {
		j.Path = r.Path
	}
	if len(j.Code) == 0 {
		j.Code = r.Code
	}
	if len(j.Title) == 0 {
		j.Title = r.Title
	}
	if len(j.Cover) == 0 {
		j.Cover = r.Cover
	}
	if j.Score == 0 {
		j.Score = r.Score
	}
	if j.ScoreCount == 0 {
		j.ScoreCount = r.ScoreCount
	}
	if j.PubDate.IsZero() {
		j.PubDate = r.PubDate
	}
	if !j.HasZH {
		j.HasZH = r.HasZH
	}

	return nil
}

func (j *JavDB) loadReviews() error {
	if j.deleted {
		return nil
	}
	r, err := j.req.requestReviews()
	if err != nil {
		return err
	}
	j.Reviews = r.Reviews

	return nil
}
