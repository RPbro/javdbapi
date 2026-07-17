package scrape

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	wantCountPattern    = regexp.MustCompile(`([0-9,]+)人(?:想看|想要)`)
	watchedCountPattern = regexp.MustCompile(`([0-9,]+)人(?:看過|看过)`)
)

func ParseDetail(doc *goquery.Document, id string) (Result[Detail], error) {
	result := Result[Detail]{}

	id = strings.TrimSpace(id)
	if id == "" {
		return result, fmt.Errorf("%w: detail missing video id", ErrParse)
	}

	title := strings.TrimSpace(doc.Find(".current-title").First().Text())
	if title == "" {
		return result, fmt.Errorf("%w: detail missing title", ErrParse)
	}
	coverURL, _ := doc.Find(".video-cover").Attr("src")

	var (
		code            string
		publishedAt     time.Time
		sawDate         bool
		score           *Score
		durationMinutes *int
		director        *NamedRef
		maker           *NamedRef
		series          *NamedRef
		actors          []Actor
		tags            []Tag
		warnings        []Warning
		panelErr        error
	)

	doc.Find(selectorDetailPanel).EachWithBreak(func(_ int, block *goquery.Selection) bool {
		label := strings.TrimSpace(block.Find("strong").First().Text())
		valueSel := block.Find(".value").First()

		switch label {
		case labelCode:
			if clip, ok := valueSel.Find("[data-clipboard-text]").Attr("data-clipboard-text"); ok && strings.TrimSpace(clip) != "" {
				code = strings.ToUpper(strings.TrimSpace(clip))
			} else {
				code = strings.ToUpper(strings.TrimSpace(valueSel.Text()))
			}
		case labelDate:
			sawDate = true
			value := strings.TrimSpace(valueSel.Text())
			if value == "" {
				panelErr = fmt.Errorf("%w: detail missing date", ErrParse)
				return false
			}
			dt, err := time.Parse(listDateLayout, value)
			if err != nil {
				panelErr = fmt.Errorf("%w: invalid detail date %q: %w", ErrParse, value, err)
				return false
			}
			publishedAt = dt
		case labelScore:
			value := strings.TrimSpace(valueSel.Text())
			if value == "" {
				warnings = append(warnings, Warning{Field: "score", Message: "empty score value"})
			} else if s, err := parseScore(value); err != nil {
				warnings = append(warnings, Warning{Field: "score", Message: err.Error()})
			} else {
				score = &s
			}
		case labelDuration:
			value := strings.TrimSpace(valueSel.Text())
			if value != "" {
				minutes, err := parseCount(value)
				if err != nil {
					warnings = append(warnings, Warning{Field: "duration_minutes", Message: err.Error()})
				} else {
					durationMinutes = &minutes
				}
			}
		case labelDirector:
			ref, err := namedRefFromPanel(valueSel, "directors")
			if err != nil {
				warnings = append(warnings, Warning{Field: "director", Message: err.Error()})
			} else {
				director = ref
			}
		case labelMaker:
			ref, err := namedRefFromPanel(valueSel, "makers")
			if err != nil {
				warnings = append(warnings, Warning{Field: "maker", Message: err.Error()})
			} else {
				maker = ref
			}
		case labelSeries:
			ref, err := namedRefFromPanel(valueSel, "series")
			if err != nil {
				warnings = append(warnings, Warning{Field: "series", Message: err.Error()})
			} else {
				series = ref
			}
		case labelTags:
			valueSel.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
				name := strings.TrimSpace(a.Text())
				href, _ := a.Attr("href")
				if name == "" || strings.TrimSpace(href) == "" {
					return
				}
				tags = append(tags, Tag{Name: name, Path: strings.TrimSpace(href)})
			})
		case labelActors:
			// Actor links and their gender symbol are flat siblings inside
			// .value (no per-actor wrapper element), so gender is read off
			// the symbol immediately following each actor's anchor.
			valueSel.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
				href, ok := a.Attr("href")
				if !ok {
					return
				}
				actorID, err := refIDFromHref(href, "actors")
				if err != nil {
					warnings = append(warnings, Warning{Field: "actors", Message: err.Error()})
					return
				}
				gender := GenderUnknown
				if sym := a.Next(); sym.HasClass("symbol") {
					switch {
					case sym.HasClass("female"):
						gender = GenderFemale
					case sym.HasClass("male"):
						gender = GenderMale
					}
				}
				actors = append(actors, Actor{ID: actorID, Name: strings.TrimSpace(a.Text()), Gender: gender})
			})
		}
		return true
	})
	if panelErr != nil {
		return result, panelErr
	}
	if code == "" {
		return result, fmt.Errorf("%w: detail missing code", ErrParse)
	}
	if !sawDate {
		return result, fmt.Errorf("%w: detail missing date", ErrParse)
	}

	// Want/watched counts share a single free-form text node, e.g.
	// "14879人想看, 3080人看過" — there is no dedicated element per count.
	metaText := doc.Find(".video-meta-panel").First().Text()
	wantCount, wantErr := parseOptionalCountFromText(metaText, wantCountPattern)
	if wantErr != nil {
		warnings = append(warnings, Warning{Field: "want_count", Message: wantErr.Error()})
	}
	watchedCount, watchedErr := parseOptionalCountFromText(metaText, watchedCountPattern)
	if watchedErr != nil {
		warnings = append(warnings, Warning{Field: "watched_count", Message: watchedErr.Error()})
	}

	screenshots := []Screenshot{}
	doc.Find(".preview-images .tile-item").Each(func(_ int, a *goquery.Selection) {
		fullURL, ok := a.Attr("href")
		if !ok || strings.TrimSpace(fullURL) == "" {
			return
		}
		thumbnailURL, _ := a.Find("img").Attr("src")
		screenshots = append(screenshots, Screenshot{
			FullURL:      strings.TrimSpace(fullURL),
			ThumbnailURL: strings.TrimSpace(thumbnailURL),
		})
	})

	magnets := []Magnet{}
	doc.Find(selectorMagnetItem).Each(func(_ int, m *goquery.Selection) {
		magnet, magnetWarnings, ok := parseMagnetItem(m)
		warnings = append(warnings, magnetWarnings...)
		if ok {
			magnets = append(magnets, magnet)
		}
	})

	hasSubtitleMagnet := false
	for _, m := range magnets {
		if m.HasSubtitle {
			hasSubtitleMagnet = true
			break
		}
	}

	if actors == nil {
		actors = []Actor{}
	}
	if tags == nil {
		tags = []Tag{}
	}

	result.Value = Detail{
		Summary: Summary{
			ID:          id,
			Code:        code,
			Title:       title,
			CoverURL:    strings.TrimSpace(coverURL),
			PublishedAt: publishedAt,
			Score:       score,
			Availability: Availability{
				HasMagnet:         len(magnets) > 0,
				HasSubtitleMagnet: hasSubtitleMagnet,
				IsPlayable:        doc.Find(selectorPlayButton).Length() > 0,
			},
		},
		DurationMinutes: durationMinutes,
		Director:        director,
		Maker:           maker,
		Series:          series,
		Actors:          actors,
		Tags:            tags,
		Screenshots:     screenshots,
		Magnets:         magnets,
		WantCount:       wantCount,
		WatchedCount:    watchedCount,
	}
	result.Warnings = warnings
	return result, nil
}

func namedRefFromPanel(valueSel *goquery.Selection, routeSegment string) (*NamedRef, error) {
	a := valueSel.Find("a[href]").First()
	href, ok := a.Attr("href")
	if !ok {
		return nil, nil
	}
	id, err := refIDFromHref(href, routeSegment)
	if err != nil {
		return nil, err
	}
	return &NamedRef{ID: id, Name: strings.TrimSpace(a.Text())}, nil
}

func parseOptionalCountFromText(text string, pattern *regexp.Regexp) (*int, error) {
	m := pattern.FindStringSubmatch(text)
	if m == nil {
		return nil, nil
	}
	n, err := parseCount(m[1])
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func parseMagnetItem(m *goquery.Selection) (Magnet, []Warning, bool) {
	var warnings []Warning

	nameAnchor := m.Find(".magnet-name a").First()
	name := strings.TrimSpace(m.Find(".magnet-name .name").First().Text())

	uri := ""
	if clip, ok := m.Find("[data-clipboard-text]").First().Attr("data-clipboard-text"); ok && strings.TrimSpace(clip) != "" {
		uri = strings.TrimSpace(clip)
	} else if href, ok := nameAnchor.Attr("href"); ok {
		uri = strings.TrimSpace(href)
	}
	if name == "" || uri == "" {
		return Magnet{}, []Warning{{Field: "magnets", Message: "missing magnet name or uri"}}, false
	}

	metaText := strings.TrimSpace(m.Find(".meta").Text())
	meta, err := parseMagnetMeta(metaText)
	if err != nil {
		warnings = append(warnings, Warning{Field: "magnets.size", Message: err.Error()})
		meta = magnetMeta{SizeText: normalizeSpace(metaText)}
	}

	tagText := m.Find(".tags").Text()
	isHD := strings.Contains(tagText, "高清") || strings.Contains(strings.ToUpper(tagText), "HD")
	hasSubtitle := strings.Contains(tagText, "中字") || strings.Contains(tagText, "字幕")

	var publishedAt *time.Time
	dateText := strings.TrimSpace(m.Find(".date .time").Text())
	if dateText != "" {
		dt, err := time.Parse(listDateLayout, dateText)
		if err != nil {
			warnings = append(warnings, Warning{Field: "magnets.published_at", Message: err.Error()})
		} else {
			publishedAt = &dt
		}
	}

	return Magnet{
		URI:         uri,
		Name:        name,
		SizeText:    meta.SizeText,
		SizeBytes:   meta.SizeBytes,
		FileCount:   meta.FileCount,
		HasSubtitle: hasSubtitle,
		IsHD:        isHD,
		PublishedAt: publishedAt,
	}, warnings, true
}
