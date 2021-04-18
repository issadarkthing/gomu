package lyric

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
)

type LyricFetcherEn struct{}

// LyricFetch should receive SongTag that was returned from GetLyricOptions, and
// returns lyric of the queried song.
func (en LyricFetcherEn) LyricFetch(songTag *SongTag) (string, error) {

	var lyric string
	c := colly.NewCollector()

	c.OnHTML("span#ctl00_ContentPlaceHolder1_lbllyrics", func(e *colly.HTMLElement) {
		content, err := e.DOM.Html()
		if err != nil {
			panic(err)
		}

		lyric = cleanHTML(content)
	})

	err := c.Visit(songTag.URL + "&type=lrc")
	if err != nil {
		return "", err
	}
	if lyric == "" {
		return "", fmt.Errorf("no lyric available")
	}
	if looksLikeLRC(lyric) {
		return lyric, nil
	}
	return "", fmt.Errorf("lyric not compatible")
}

// LyricOptions queries available song lyrics. It returns slice of SongTag
func (en LyricFetcherEn) LyricOptions(search string) ([]*SongTag, error) {

	var songTags []*SongTag

	c := colly.NewCollector()

	c.OnHTML("#tablecontainer td a", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		title := strings.TrimSpace(e.Text)
		songTag := &SongTag{
			URL:           link,
			TitleForPopup: title,
			LangExt:       "en",
		}
		songTags = append(songTags, songTag)
	})

	query := url.QueryEscape(search)
	err := c.Visit("https://www.rentanadviser.com/en/subtitles/subtitles4songs.aspx?src=" + query)
	if err != nil {
		return nil, err
	}

	return songTags, err
}
