package javdbapi

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

var errorFiltered = errors.New("filtered")

type request struct {
	client *http.Client
	ua     string
	limit  int
	filter Filter
	url    string
}

func (r *request) requestList() ([]*JavDB, error) {
	reader, err := r.do()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var results []*JavDB
	var count int

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	doc.Find("div.item").Each(func(i int, selection *goquery.Selection) {
		if r.limit > 0 && count == r.limit {
			return
		}
		var (
			title, code, cover, path string
			score                    float64
			scoreCount               int
			pubDate                  time.Time
			hasZH                    bool
		)

		{
			titleArr := strings.Split(selection.Find(".video-title").Text(), " ")
			if len(titleArr) < 2 {
				return
			}
			// title
			title = strTrimSpace(strings.Join(titleArr[1:], " "))
			if utf8.RuneCountInString(title) == 0 {
				return
			}
			// code
			code = strings.ToUpper(strTrimSpace(titleArr[0]))
			if len(strings.Split(code, "-")) != 2 || !strIsInt(strings.Split(code, "-")[1]) {
				return
			}
		}

		{
			// cover
			coverStr, exists := selection.Find(".cover, .contain").First().Find("img").Attr("src")
			if !exists || len(coverStr) == 0 {
				return
			}
			cover = coverStr
		}

		{
			// link(path)
			linkPath, exists := selection.Find(".box").Attr("href")
			if !exists || len(linkPath) == 0 {
				return
			}
			path = linkPath
		}

		{
			scoreArr := strings.Split(strTrimSpace(selection.Find(".score > .value").Text()), ",")
			if len(scoreArr) != 2 {
				return
			}
			scoreText := scoreArr[0]
			scoreCountText := scoreArr[1]
			scoreTextArr := strings.Split(scoreText, "分")
			if len(scoreTextArr) != 2 {
				return
			}
			// score
			score, err = strconv.ParseFloat(scoreTextArr[0], 64)
			if err != nil {
				return
			}
			if r.filter.ScoreGT > 0 && score < r.filter.ScoreGT {
				return
			}
			if r.filter.ScoreLT > 0 && score > r.filter.ScoreLT {
				return
			}
			// scoreCount
			scoreCountTextArr := strings.Split(scoreCountText, "人評價")
			if len(scoreCountTextArr) == 2 && len(strings.Split(scoreCountTextArr[0], "由")) == 2 {
				scoreCount, err = strconv.Atoi(strings.Split(scoreCountTextArr[0], "由")[1])
				if err != nil {
					return
				}
			}
			if r.filter.ScoreCountGT > 0 && scoreCount < r.filter.ScoreCountGT {
				return
			}
			if r.filter.ScoreCountLT > 0 && scoreCount > r.filter.ScoreCountLT {
				return
			}
		}

		{
			// pubDate
			pubDateText := strTrimSpace(selection.Find(".meta").Text())
			if len(pubDateText) > 0 {
				pubDateTime, err := time.Parse("2006-01-02", pubDateText)
				if err != nil {
					return
				}
				if pubDateTime.Unix() <= 0 {
					return
				}
				pubDate = pubDateTime
			}
			if !r.filter.PubDateBefore.IsZero() && pubDate.After(r.filter.PubDateBefore) {
				return
			}
			if !r.filter.PubDateAfter.IsZero() && pubDate.Before(r.filter.PubDateAfter) {
				return
			}
		}

		{
			// hasZH
			if strTrimSpace(selection.Find(".tag, .is-warning").Text()) == "含中字磁鏈" {
				hasZH = true
			}
			if r.filter.HasZH && r.filter.HasZH != hasZH {
				return
			}
		}

		results = append(results, &JavDB{
			req:        r,
			Path:       path,
			Code:       code,
			Title:      title,
			Cover:      cover,
			Score:      score,
			ScoreCount: scoreCount,
			PubDate:    pubDate,
			HasZH:      hasZH,
		})
		count++
	})

	return results, nil
}

func (r *request) requestDetails() (*JavDB, error) {
	reader, err := r.do()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var (
		actresses, tags, pics, magnets []string
		preview                        string

		title, code, cover, path string
		score                    float64
		scoreCount               int
		pubDate                  time.Time
		hasZH                    bool
	)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	{
		// title
		title = strTrimSpace(doc.Find(".current-title").First().Text())
	}

	{
		// cover
		coverText, exists := doc.Find(".video-cover").Attr("src")
		if exists {
			cover = strTrimSpace(coverText)
		}
	}

	{
		doc.Find(".panel-block > strong").Each(func(i int, selection *goquery.Selection) {
			switch selection.Text() {
			case "番號:":
				// code
				code = strings.ToUpper(strTrimSpace(selection.Next().Text()))
				if len(strings.Split(code, "-")) != 2 || !strIsInt(strings.Split(code, "-")[1]) {
					return
				}
			case "日期:":
				// pubDate
				pubDateText := strTrimSpace(selection.Next().Text())
				if len(pubDateText) > 0 {
					pubDateTime, err := time.Parse("2006-01-02", pubDateText)
					if err != nil {
						return
					}
					if pubDateTime.Unix() <= 0 {
						return
					}
					pubDate = pubDateTime
				}
			case "評分:":
				scoreArr := strings.Split(strTrimSpace(selection.Next().Text()), ",")
				if len(scoreArr) != 2 {
					return
				}
				scoreText := scoreArr[0]
				scoreCountText := scoreArr[1]
				scoreTextArr := strings.Split(scoreText, "分")
				if len(scoreTextArr) != 2 {
					return
				}
				// score
				score, err = strconv.ParseFloat(scoreTextArr[0], 64)
				if err != nil {
					return
				}
				// scoreCount
				scoreCountTextArr := strings.Split(scoreCountText, "人評價")
				if len(scoreCountTextArr) == 2 && len(strings.Split(scoreCountTextArr[0], "由")) == 2 {
					scoreCount, err = strconv.Atoi(strings.Split(scoreCountTextArr[0], "由")[1])
					if err != nil {
						return
					}
				}
			case "演員:":
				// actresses
				selection.Parent().Find(".value > .female").Each(func(i int, selection *goquery.Selection) {
					actress := strTrimSpace(selection.Prev().Text())
					if utf8.RuneCountInString(actress) == 0 {
						return
					}
					actresses = append(actresses, actress)
				})
			case "類別:":
				// tags
				selection.Parent().Find(".value > a").Each(func(i int, selection *goquery.Selection) {
					tagText := selection.Text()
					tagText = strings.ReplaceAll(tagText, "・", "")
					tagText = strings.ReplaceAll(tagText, "，", "")
					tagText = strings.ReplaceAll(tagText, "、", "")
					if utf8.RuneCountInString(tagText) == 0 {
						return
					}
					tags = append(tags, tagText)
				})
			}
		})
		if len(actresses) == 0 || len(tags) == 0 {
			return nil, errorFiltered
		}
		if len(r.filter.ActressesIn) > 0 {
			if !sliceContainsAny(actresses, r.filter.ActressesIn) {
				return nil, errorFiltered
			}
		}
		if len(r.filter.ActressesNotIn) > 0 {
			if sliceContainsAny(actresses, r.filter.ActressesNotIn) {
				return nil, errorFiltered
			}
		}
		if len(r.filter.TagsIn) > 0 {
			if !sliceContainsAny(tags, r.filter.TagsIn) {
				return nil, errorFiltered
			}
		}
		if len(r.filter.TagsNotIn) > 0 {
			if sliceContainsAny(tags, r.filter.TagsNotIn) {
				return nil, errorFiltered
			}
		}
	}

	{
		// pics
		doc.Find(".preview-images > .tile-item").Each(func(i int, selection *goquery.Selection) {
			pic, exists := selection.Attr("href")
			if !exists {
				return
			}
			pics = append(pics, pic)
		})
		if r.filter.HasPics && len(pics) == 0 {
			return nil, errorFiltered
		}
	}

	{
		// magnets
		doc.Find("#magnets-content .magnet-name > a").Each(func(i int, selection *goquery.Selection) {
			magnet, exists := selection.Attr("href")
			if !exists || !strIsMagnet(magnet) {
				return
			}
			magnets = append(magnets, magnet)
			if strings.Contains(strTrimSpace(selection.Find(".tags").Text()), "字幕") {
				hasZH = true
			}
		})
		if r.filter.HasMagnets && len(magnets) == 0 {
			return nil, errorFiltered
		}
	}

	{
		// preview
		previewText, exists := doc.Find("#preview-video > source").First().Attr("src")
		if exists && strings.HasSuffix(previewText, "mp4") {
			if strings.Index(previewText, "//") == 0 {
				previewText = "https:" + previewText
			}
			preview = previewText
		}
		if r.filter.HasPreview && len(preview) == 0 {
			return nil, errorFiltered
		}
	}

	{
		// path
		u, err := url.Parse(r.url)
		if err != nil {
			return nil, err
		}
		path = u.Path
	}

	return &JavDB{
		Path:       path,
		Code:       code,
		Title:      title,
		Cover:      cover,
		Score:      score,
		ScoreCount: scoreCount,
		PubDate:    pubDate,
		HasZH:      hasZH,
		Preview:    preview,
		Actresses:  actresses,
		Tags:       tags,
		Pics:       pics,
		Magnets:    magnets,
	}, nil
}

func (r *request) requestReviews() (*JavDB, error) {
	reader, err := r.do()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var reviews []string

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	{
		// reviews
		doc.Find(".review-item > .content").Each(func(i int, selection *goquery.Selection) {
			review := strTrimSpace(selection.Text())
			if utf8.RuneCountInString(review) == 0 {
				return
			}
			reviews = append(reviews, review)
		})
		if r.filter.HasReviews && len(reviews) == 0 {
			return nil, errorFiltered
		}
	}

	return &JavDB{
		Reviews: reviews,
	}, nil
}

func (r *request) do() (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, r.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", r.ua)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
