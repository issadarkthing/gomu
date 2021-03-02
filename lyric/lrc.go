// Package lyric package download lyrics from different website and embed them into mp3 file.
// lrc file is used to parse lrc file into subtitle. Similar to subtitles package
package lyric

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Lyric struct {
	Captions []Caption
}

type Caption struct {
	Seq   int
	Start time.Time
	End   time.Time
	Text  []string
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
	endString := "[158:00.00]The End" + eol
	s = s + endString
	s = cleanLRC(s)
	r1 := regexp.MustCompile(`(?U)^\[[0-9].*\]`)
	lines := strings.Split(s, "\n")
	outSeq := 1

	for i := 0; i < len(lines)-1; i++ {
		seq := strings.Trim(lines[i], "\r ")
		if seq == "" {
			continue
		}

		var matchEnd []string
		matchStart := r1.FindStringSubmatch(lines[i])
		matchEnd = r1.FindStringSubmatch(lines[i+1])

		if len(matchStart) < 1 || len(matchEnd) < 1 {
			// err = fmt.Errorf("lrc: parse error at line %d (idx out of range) for input '%s'", i, lines[i])
			// break
			// Here we continue to parse the subtitle and ignore the lines have no start or end
			continue
		}

		var o Caption
		o.Seq = outSeq

		o.Start, err = parseLrcTime(matchStart[0])
		if err != nil {
			err = fmt.Errorf("lrc: start error at line %d: %v", i, err)
			break
		}

		o.End, err = parseLrcTime(matchEnd[0])
		if err != nil {
			err = fmt.Errorf("lrc: end error at line %d: %v", i, err)
			break
		}

		r2 := regexp.MustCompile(`^\[.*\]`)
		s2 := r2.ReplaceAllString(lines[i], "$1")
		s3 := strings.Trim(s2, "\r ")
		o.Text = append(o.Text, s3)
		// Seems that empty lines are useful and shouldn't be deleted
		// if len(o.Text) > 0 {
		res.Captions = append(res.Captions, o)
		outSeq++
		// }
	}
	return
}

// parseSrtTime parses a lrc subtitle time (duration since start of film)
func parseLrcTime(in string) (time.Time, error) {
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
		return time.Now(), fmt.Errorf("[lrc] Regexp didnt match: %s", in)
	}
	h := 0
	m, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Now(), err
	}
	s, err := strconv.Atoi(matches[2])
	if err != nil {
		return time.Now(), err
	}
	ms, err := strconv.Atoi(matches[3])
	if err != nil {
		return time.Now(), err
	}

	return makeTime(h, m, s, ms), nil
}

// makeTime is a helper to create a time duration
func makeTime(h int, m int, s int, ms int) time.Time {
	return time.Date(0, 1, 1, h, m, s, ms*1000*1000, time.UTC)
}

// cleanLRC clean the string download
func cleanLRC(s string) (cleanLyric string) {
	// Clean &apos; to '
	s = strings.ToValidUTF8(s, " ")
	s = strings.Replace(s, "&apos;", "'", -1)
	// It's wierd that sometimes there are two ajacent ''.
	// Replace it anyway
	cleanLyric = strings.Replace(s, "''", "'", -1)

	return cleanLyric
}

// AsLRC renders the sub in .srt format
func (lyric Lyric) AsLRC() (res string) {
	for _, sub := range lyric.Captions {
		res += sub.AsLRC()
	}
	return
}

// AsLRC renders the caption as srt
func (cap Caption) AsLRC() string {
	// res := fmt.Sprintf("%d", cap.Caption.Seq) + eol +
	// 	TimeLRC(cap.Caption.Start) + " --> " + TimeLRC(cap.Caption.End) + eol
	res := "[" + TimeLRC(cap.Start) + "]"
	for _, line := range cap.Text {
		res += line + eol
	}
	return res
}

// TimeLRC renders a timestamp for use in .srt
func TimeLRC(t time.Time) string {
	res := t.Format("04:05.00")
	// return strings.Replace(res, ".", ",", 1)
	return res
}

//ResyncSubs can ajust delay of lyrics
func (lyric *Lyric) ResyncSubs(sync int) {
	for i := range lyric.Captions {
		lyric.Captions[i].Start = lyric.Captions[i].Start.
			Add(time.Duration(sync) * time.Millisecond)
		lyric.Captions[i].End = lyric.Captions[i].End.
			Add(time.Duration(sync) * time.Millisecond)
	}
}
