package scrape

const (
	selectorListItem           = "div.item"
	selectorNextPage           = "a.pagination-next[rel=next]"
	selectorDetailPanel        = ".movie-panel-info .panel-block"
	selectorReviewItem         = ".review-item[id^='review-item-']"
	selectorEmptyState         = ".empty-message"
	selectorReviewsMessageBody = "article.message.video-panel > .message-body"
	selectorMagnetItem         = "#magnets-content > .item.columns.is-desktop"
	selectorPlayButton         = ".video-detail .play-button"
)

// reviewsEmptyMessageText is the exact text javdb.com renders inside
// selectorReviewsMessageBody when a video has no reviews. Matching on this
// text (not just the selector) keeps the check narrow: an unrelated message
// box using the same "article.message.video-panel" markup — e.g. a site
// notice or error banner — must never be misread as an empty reviews page.
const reviewsEmptyMessageText = "暫無內容"

const (
	labelCode     = "番號:"
	labelDate     = "日期:"
	labelDuration = "時長:"
	labelDirector = "導演:"
	labelMaker    = "片商:"
	labelSeries   = "系列:"
	labelScore    = "評分:"
	labelTags     = "類別:"
	labelActors   = "演員:"
)
