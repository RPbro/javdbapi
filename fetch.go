package javdbapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

type Item struct {
	remove bool

	ID string // path

	Title       string
	Code        string
	Cover       string
	Path        string
	Score       float64
	ScoreCount  int
	PubDate     time.Time
	HasSubtitle bool

	Preview string
	Actors  []string
	Tags    []string
	Pics    []string
	Magnets []string

	Reviews []string
}

func (a *API) fetchList(hc *http.Client, link string) ([]*Item, error) {
	var result []*Item

	doc, err := a.toDocument(hc, link)
	if err != nil {
		return nil, err
	}

	doc.Find("div.item").Each(func(i int, selection *goquery.Selection) {
		var (
			title, code, cover, path string
			score                    float64
			scoreCount               int
			pubDate                  time.Time
			hasSubtitle              bool
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
			// path
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
			// hasSubtitle
			if strTrimSpace(selection.Find(".tag, .is-warning").Text()) == "含中字磁鏈" {
				hasSubtitle = true
			}
		}

		if a.filter != nil {
			a.filter.pass = true
			a.filter.checkScoreGT(score)
			a.filter.checkScoreLT(score)
			a.filter.checkScoreCountGT(scoreCount)
			a.filter.checkScoreCountLT(scoreCount)
			a.filter.checkPubDateBefore(pubDate)
			a.filter.checkPubDateAfter(pubDate)
			a.filter.checkHasSubtitle(hasSubtitle)
			if !a.filter.pass {
				return
			}
		}

		result = append(result, &Item{
			ID:          path,
			Title:       title,
			Code:        code,
			Cover:       cover,
			Path:        path,
			Score:       score,
			ScoreCount:  scoreCount,
			PubDate:     pubDate,
			HasSubtitle: hasSubtitle,
		})
	})

	return result, nil
}

func (a *API) fetchDetail(hc *http.Client, link string, item *Item) (*Item, error) {
	doc, err := a.toDocument(hc, link)
	if err != nil {
		return nil, err
	}

	if item == nil {
		var (
			title, code, cover, path string
			score                    float64
			scoreCount               int
			pubDate                  time.Time
			hasSubtitle              bool
		)

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
			// path
			u, err := url.Parse(link)
			if err != nil {
				return nil, err
			}
			path = u.Path
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
				}
			})
		}

		item = &Item{
			ID:          path,
			Title:       title,
			Code:        code,
			Cover:       cover,
			Path:        path,
			Score:       score,
			ScoreCount:  scoreCount,
			PubDate:     pubDate,
			HasSubtitle: hasSubtitle,
		}
	}

	var (
		preview                     string
		actors, tags, pics, magnets []string
	)

	// preview
	previewText, exists := doc.Find("#preview-video > source").First().Attr("src")
	if exists && strings.HasSuffix(previewText, "mp4") {
		if strings.Index(previewText, "//") == 0 {
			previewText = "https:" + previewText
		}
		preview = previewText
	}

	doc.Find(".panel-block > strong").Each(func(i int, selection *goquery.Selection) {
		switch selection.Text() {
		case "演員:":
			// actors
			selection.Parent().Find(".value > .female").Each(func(i int, selection *goquery.Selection) {
				actor := strTrimSpace(selection.Prev().Text())
				if utf8.RuneCountInString(actor) == 0 {
					return
				}
				actors = append(actors, actor)
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

	// pics
	doc.Find(".preview-images > .tile-item").Each(func(i int, selection *goquery.Selection) {
		pic, exists := selection.Attr("href")
		if !exists {
			return
		}
		pics = append(pics, pic)
	})

	// magnets
	doc.Find("#magnets-content .magnet-name > a").Each(func(i int, selection *goquery.Selection) {
		magnet, exists := selection.Attr("href")
		if !exists || !strIsMagnet(magnet) {
			return
		}
		magnets = append(magnets, magnet)
	})

	if a.filter != nil {
		a.filter.pass = true
		a.filter.checkActorsIn(actors)
		a.filter.checkActorsNotIn(actors)
		a.filter.checkTagsIn(tags)
		a.filter.checkTagsNotIn(tags)
		a.filter.checkHasPics(pics)
		a.filter.checkHasMagnets(magnets)
		a.filter.checkHasPreview(preview)
		if !a.filter.pass {
			item.remove = true
		}
	}

	item.Preview = preview
	item.Actors = actors
	item.Tags = tags
	item.Pics = pics
	item.Magnets = magnets

	return item, nil
}

func (a *API) fetchReviews(hc *http.Client, link string, item *Item) (*Item, error) {
	doc, err := a.toDocument(hc, link)
	if err != nil {
		return nil, err
	}

	var reviews []string

	// reviews
	doc.Find(".review-item > .content").Each(func(i int, selection *goquery.Selection) {
		review := strTrimSpace(selection.Text())
		if utf8.RuneCountInString(review) == 0 {
			return
		}
		reviews = append(reviews, review)
	})

	if a.filter != nil {
		a.filter.pass = true
		a.filter.checkHasReviews(reviews)
		if !a.filter.pass {
			item.remove = true
		}
	}

	item.Reviews = reviews

	return item, nil
}

func (a *API) toDocument(hc *http.Client, link string) (*goquery.Document, error) {
	resp, err := hc.Get(link)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return doc, nil
}
