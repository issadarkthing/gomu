package lyric

import (
	"html"
	"regexp"
	"strings"
)

type SongTag struct {
	Artist          string
	Title           string
	Album           string
	URL             string
	TitleForPopup   string
	LangExt         string
	ServiceProvider string
	SongID          string // SongID and LyricID is returned by cn server. It's not guaranteed to be identical
	LyricID         string
}

// GetLyric should receive url that was returned from GetLyricOptions. GetLyric
// returns lyric of the queried song.
func GetLyric(lang string, songTag *SongTag) (string, error) {

	switch lang {
	case "en":
		return getLyricEn(songTag)
	case "zh-CN":
		return getLyricCn(songTag)
	default:
		return getLyricEn(songTag)
	}

}

// GetLyricOptions queries available song lyrics. It returns map of title and
// url of the lyric.
func GetLyricOptions(lang string, search string) ([]*SongTag, error) {

	switch lang {
	case "en":
		return getLyricOptionsEn(search)
	case "zh-CN":
		return getLyricOptionsCn(search)
	default:
		return getLyricOptionsEn(search)
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
