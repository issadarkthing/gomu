package lyric

import (
	"html"
	"regexp"
	"strings"
)

// SongTag is the tag information for songs
type SongTag struct {
	Artist          string
	Title           string
	Album           string
	URL             string
	TitleForPopup   string
	LangExt         string
	ServiceProvider string
	SongID          string // SongID and LyricID is returned from cn server. It's not guaranteed to be identical
	LyricID         string
}

// LyricFetcher is the interface to get lyrics via different language
type LyricFetcher interface {
	LyricFetch(songTag *SongTag) (string, error)
	LyricOptions(search string) ([]*SongTag, error)
}

func LyricLang(lang string) LyricFetcher {

	switch lang {
	case "en":
		return LyricFetcherEn{}
	case "zh-CN":
		return LyricFetcherCn{}
	default:
		return LyricFetcherEn{}
	}

}

// cleanHTML parses html text to valid utf-8 text
func cleanHTML(input string) string {

	content := html.UnescapeString(input)
	// delete heading tag
	re := regexp.MustCompile(`^<h3>.*`)
	content = re.ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "\r\n", "")
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	// remove non-utf8 character
	re = regexp.MustCompile(`‚`)
	content = re.ReplaceAllString(content, ",")
	content = strings.ToValidUTF8(content, " ")
	content = strings.Map(func(r rune) rune {
		if r == 160 {
			return 32
		}
		return r
	}, content)

	return content
}
