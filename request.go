package javdbapi

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

type request struct {
	client *http.Client
	ua     string
	limit  int
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
			// scoreCount
			scoreCountTextArr := strings.Split(scoreCountText, "人評價")
			if len(scoreCountTextArr) == 2 && len(strings.Split(scoreCountTextArr[0], "由")) == 2 {
				scoreCount, err = strconv.Atoi(strings.Split(scoreCountTextArr[0], "由")[1])
				if err != nil {
					return
				}
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
		}

		{
			// hasZH
			if strTrimSpace(selection.Find(".tag, .is-warning").Text()) == "含中字磁鏈" {
				hasZH = true
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
	)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	{
		doc.Find(".panel-block > strong").Each(func(i int, selection *goquery.Selection) {
			switch selection.Text() {
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
					if utf8.RuneCountInString(tagText) == 0 {
						return
					}
					tags = append(tags, tagText)
				})
			}
		})
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
	}

	{
		// magnets
		doc.Find("#magnets-content .magnet-name > a").Each(func(i int, selection *goquery.Selection) {
			magnet, exists := selection.Attr("href")
			if !exists || !strIsMagnet(magnet) {
				return
			}
			magnets = append(magnets, magnet)
		})
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
	}

	return &JavDB{
		Preview:   preview,
		Actresses: actresses,
		Tags:      tags,
		Pics:      pics,
		Magnets:   magnets,
	}, nil
}

func (r *request) requestReviews() (*JavDB, error) {
	reader, err := r.do()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	var reviews []string
	{
		doc.Find(".review-item > .content").Each(func(i int, selection *goquery.Selection) {
			review := strTrimSpace(selection.Text())
			if utf8.RuneCountInString(review) == 0 {
				return
			}
			reviews = append(reviews, review)
		})
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
