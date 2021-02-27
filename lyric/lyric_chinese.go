package lyric

import (
	// "encoding/json"
	"fmt"
	// "net/url"
	"strconv"

	"github.com/asmcos/requests"
	"github.com/martinlindhe/subtitles"
	// "github.com/gocolly/colly"
	// r "github.com/solos/requests"
)

// GetLyricOptionsChinese queries available song lyrics. It returns map of title and
// id of the lyric.
func GetLyricOptionsChinese(search string) (map[string]string, error) {

	result := make(map[string]string)
	p := requests.Params{
		"site":   "netease",
		"search": search,
	}
	req := requests.Requests()
	req.Header.Set("Content-Type", "application/json")
	resp, err := req.Get("http://api.sunyj.xyz", p)
	if err != nil {
		return nil, err
	}

	var dataMap []map[string]interface{}
	err = resp.Json(&dataMap)
	if err != nil {
		return nil, err
	}
	for k, v := range dataMap {
		lyricIDfloat64 := dataMap[k]["lyric_id"]
		songName := v["name"]
		songArtist := v["artist"]
		lyricID := strconv.FormatFloat(lyricIDfloat64.(float64), 'f', -1, 64)
		songTitle := fmt.Sprintf("%s - %s ", songArtist, songName)
		if lyricID == "" {
			continue
		}
		result[songTitle] = lyricID
	}

	return result, nil
}

// GetLyricChinese should receive url that was returned from GetLyricOptions. GetLyric
// returns lyric of the queried song.
func GetLyricChinese(lyricID string) (string, error) {

	var lyric string
	p := requests.Params{
		"site":  "netease",
		"lyric": lyricID,
	}
	req := requests.Requests()
	resp, err := req.Get("http://api.sunyj.xyz", p)
	if err != nil {
		return "", err
	}
	var dataMap map[string]interface{}
	err = resp.Json(&dataMap)
	if err != nil {
		return "", err
	}
	lyric = dataMap["lyric"].(string)
	if looksLikeLRC(lyric) {
		// err = fmt.Errorf("is a lrc file and need to convert to srt")
		// return "", err
		var tmpSubtitle *subtitles.Subtitle
		tmpSubtitle, err = NewFromLRC(lyric)
		if err != nil {
			return "", err
		}
		lyric = tmpSubtitle.AsSRT()
	}
	return lyric, nil
}
