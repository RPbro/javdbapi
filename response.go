package javdbapi

import (
	"time"
)

type JavDB struct {
	req *request

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
	r, err := j.req.requestDetails()
	if err != nil {
		return err
	}
	j.Preview = r.Preview
	j.Actresses = r.Actresses
	j.Tags = r.Tags
	j.Pics = r.Pics
	j.Magnets = r.Magnets

	return nil
}

func (j *JavDB) loadReviews() error {
	r, err := j.req.requestReviews()
	if err != nil {
		return err
	}
	j.Reviews = r.Reviews

	return nil
}
