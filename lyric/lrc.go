// Package lyric package download lyrics from different website and embed them into mp3 file.
// lrc file is used to parse lrc file into subtitle. Similar to subtitles package
// [al:''Album where the song is from'']
// [ar:''Lyrics artist'']
// [by:''Creator of the LRC file'']
// [offset:''+/- Overall timestamp adjustment in milliseconds, + shifts time up, - shifts down'']
// [re:''The player or editor that creates LRC file'']
// [ti:''Lyrics (song) title'']
// [ve:''version of program'']
// [ti:Let's Twist Again]
// [ar:Chubby Checker oppure  Beatles, The]
// [au:Written by Kal Mann / Dave Appell, 1961]
// [al:Hits Of The 60's - Vol. 2 – Oldies]
// [00:12.00]Lyrics beginning ...
// [00:15.30]Some more lyrics ...
package lyric

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/ztrue/tracerr"
)

type Lyric struct {
	Album               string
	Artist              string
	ByCreator           string // Creator of LRC file
	Offset              int32  // positive means delay lyric
	RePlayerEditor      string // Player or Editor to create this LRC file
	Title               string
	VersionPlayerEditor string // Version of player or editor
	LangExt             string
	UnsyncedCaptions    []UnsyncedCaption  // USLT captions
	SyncedCaptions      []id3v2.SyncedText // SYLT captions
}

type UnsyncedCaption struct {
	Timestamp uint32
	Text      string
}

// Eol is the end of line characters to use when writing .srt data
var eol = "\n"

func init() {
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}
}

func looksLikeLRC(s string) bool {
	if s != "" {
		if s[0] == 239 || s[0] == 91 {
			return true
		}
	}
	return false
}

// NewFromLRC parses a .lrc text into Subtitle, assumes s is a clean utf8 string
func NewFromLRC(s string) (res Lyric, err error) {
	s = cleanLRC(s)
	lines := strings.Split(s, "\n")

	for i := 0; i < len(lines)-1; i++ {
		seq := strings.Trim(lines[i], "\r ")
		if seq == "" {
			continue
		}

		if strings.HasPrefix(seq, "[offset") {
			tmpString := strings.TrimPrefix(seq, "[offset:")
			tmpString = strings.TrimSuffix(tmpString, "]")
			tmpString = strings.ReplaceAll(tmpString, " ", "")
			var intOffset int
			intOffset, err = strconv.Atoi(tmpString)
			if err != nil {
				return res, tracerr.Wrap(err)
			}
			res.Offset = int32(intOffset)
		}

		timestampPattern := regexp.MustCompile(`(?U)^\[[0-9].*\]`)
		matchTimestamp := timestampPattern.FindStringSubmatch(lines[i])

		if len(matchTimestamp) < 1 {
			// Here we continue to parse the subtitle and ignore the lines have no timestamp
			continue
		}

		var o UnsyncedCaption

		o.Timestamp, err = parseLrcTime(matchTimestamp[0])
		if err != nil {
			err = fmt.Errorf("lrc: start error at line %d: %v", i, err)
			break
		}

		r2 := regexp.MustCompile(`^\[.*\]`)
		s2 := r2.ReplaceAllString(lines[i], "$1")
		s3 := strings.Trim(s2, "\r")
		s3 = strings.Trim(s3, "\n")
		s3 = strings.TrimSpace(s3)
		singleSpacePattern := regexp.MustCompile(`\s+`)
		s3 = singleSpacePattern.ReplaceAllString(s3, " ")
		o.Text = s3
		res.UnsyncedCaptions = append(res.UnsyncedCaptions, o)
	}

	// we sort the cpations by Timestamp. This is to fix some lyrics downloaded are not sorted
	sort.SliceStable(res.UnsyncedCaptions, func(i, j int) bool {
		return res.UnsyncedCaptions[i].Timestamp < res.UnsyncedCaptions[j].Timestamp
	})

	res = mergeLRC(res)

	// add synced lyric by calculating offset of unsynced lyric
	for _, v := range res.UnsyncedCaptions {
		var s id3v2.SyncedText
		s.Text = v.Text
		if res.Offset >= 0 {
			s.Timestamp = v.Timestamp + uint32(res.Offset)
		} else {
			if v.Timestamp > uint32(-res.Offset) {
				s.Timestamp = v.Timestamp - uint32(-res.Offset)
			} else {
				s.Timestamp = 0
			}
		}
		res.SyncedCaptions = append(res.SyncedCaptions, s)
	}

	// merge again because timestamp 0 could overlap if offset is negative
	res = mergeSyncLRC(res)
	return
}

// parseLrcTime parses a lrc subtitle time (ms since start of song)
func parseLrcTime(in string) (uint32, error) {
	in = strings.TrimPrefix(in, "[")
	in = strings.TrimSuffix(in, "]")
	// . and , to :
	in = strings.Replace(in, ",", ":", -1)
	in = strings.Replace(in, ".", ":", -1)

	if strings.Count(in, ":") == 2 {
		in += ":000"
	}

	r1 := regexp.MustCompile("([0-9]+):([0-9]+):([0-9]+):([0-9]+)")
	matches := r1.FindStringSubmatch(in)
	if len(matches) < 5 {
		return 0, fmt.Errorf("[lrc] Regexp didnt match: %s", in)
	}
	m, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	s, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, err
	}
	ms, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, err
	}

	timeStamp := m*60*1000 + s*1000 + ms
	if timeStamp < 0 {
		timeStamp = 0
	}

	return uint32(timeStamp), nil
}

// cleanLRC clean the string download
func cleanLRC(s string) (cleanLyric string) {
	// Clean &apos; to '
	s = strings.ToValidUTF8(s, " ")
	s = strings.Replace(s, "&apos;", "'", -1)
	// It's weird that sometimes there are two adjacent ''.
	// Replace it anyway
	cleanLyric = strings.Replace(s, "''", "'", -1)

	return cleanLyric
}

// mergeLRC merge lyric if the time between two captions is less than 2 seconds
func mergeLRC(lyric Lyric) (res Lyric) {

	lenLyric := len(lyric.UnsyncedCaptions)
	for i := 0; i < lenLyric-1; i++ {
		if lyric.UnsyncedCaptions[i].Timestamp+2000 > lyric.UnsyncedCaptions[i+1].Timestamp && lyric.UnsyncedCaptions[i].Text != "" {
			lyric.UnsyncedCaptions[i].Text = lyric.UnsyncedCaptions[i].Text + " " + lyric.UnsyncedCaptions[i+1].Text
			lyric.UnsyncedCaptions = removeUnsynced(lyric.UnsyncedCaptions, i+1)
			i--
			lenLyric--
		}
	}
	return lyric
}

// mergeSyncLRC merge lyric if the time between two captions is less than 2 seconds
// this is specially useful when offset is negative and several timestamp 0 in synced lyric
func mergeSyncLRC(lyric Lyric) (res Lyric) {

	lenLyric := len(lyric.SyncedCaptions)
	for i := 0; i < lenLyric-1; i++ {
		if lyric.SyncedCaptions[i].Timestamp+2000 > lyric.SyncedCaptions[i+1].Timestamp && lyric.SyncedCaptions[i].Text != "" {
			lyric.SyncedCaptions[i].Text = lyric.SyncedCaptions[i].Text + " " + lyric.SyncedCaptions[i+1].Text
			lyric.SyncedCaptions = removeSynced(lyric.SyncedCaptions, i+1)
			i--
			lenLyric--
		}
	}
	return lyric
}

func removeUnsynced(slice []UnsyncedCaption, s int) []UnsyncedCaption {
	return append(slice[:s], slice[s+1:]...)
}

func removeSynced(slice []id3v2.SyncedText, s int) []id3v2.SyncedText {
	return append(slice[:s], slice[s+1:]...)
}

// AsLRC renders the sub in .lrc format
func (lyric Lyric) AsLRC() (res string) {
	if lyric.Offset != 0 {
		stringOffset := strconv.Itoa(int(lyric.Offset))
		res += "[offset:" + stringOffset + "]" + eol
	}

	for _, sub := range lyric.UnsyncedCaptions {
		res += sub.asLRC()
	}
	return
}

// asLRC renders the caption as one line in lrc
func (cap UnsyncedCaption) asLRC() string {
	res := "[" + timeLRC(cap.Timestamp) + "]"
	res += cap.Text + eol
	return res
}

// timeLRC renders a timestamp for use in lrc
func timeLRC(t uint32) string {
	tDuration := time.Duration(t) * time.Millisecond
	h := tDuration / time.Hour
	tDuration -= h * time.Hour
	m := tDuration / time.Minute
	tDuration -= m * time.Minute
	s := tDuration / time.Second
	tDuration -= s * time.Second
	ms := tDuration / time.Millisecond

	res := fmt.Sprintf("%02d:%02d.%03d", m, s, ms)
	return res
}
