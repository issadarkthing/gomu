//Package lyric package download lyrics from different website and embed them into mp3 file.
//lrc file is used to parse lrc file into subtitle. Similar to subtitles package
package lyric

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/martinlindhe/subtitles"
)

// Eol is the end of line characters to use when writing .srt data
var eol = "\n"

func init() {
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}
}

func looksLikeLRC(s string) bool {
	return s[0] == '['
}

// NewFromLRC parses a .lrc text into Subtitle, assumes s is a clean utf8 string
func NewFromLRC(s string) (res *subtitles.Subtitle, err error) {
	r1 := regexp.MustCompile(`^\[[0-9].*\]`)
	lines := strings.Split(s, "\n")
	outSeq := 1

	for i := 0; i < len(lines); i++ {
		seq := strings.Trim(lines[i], "\r ")
		// fmt.Println(seq)
		if seq == "" {
			continue
		}

		var matchEnd []string
		matchStart := r1.FindStringSubmatch(lines[i])
		if i+1 < len(lines) {
			matchEnd = r1.FindStringSubmatch(lines[i+1])
		} else {
			matchEnd = matchStart
		}

		if len(matchStart) < 1 || len(matchEnd) < 1 {
			// err = fmt.Errorf("lrc: parse error at line %d (idx out of range) for input '%s'", i, lines[i])
			// break
			continue
		}

		var o subtitles.Caption
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

		res.Captions = append(res.Captions, o)
		outSeq++
		// i++
		// if i >= len(lines) {
		// 	break
		// }

		// textLine := 1
		// for {
		// 	line := strings.Trim(lines[i], "\r ")
		// 	if line == "" && textLine > 1 {
		// 		break
		// 	}
		// 	if line != "" {
		// 		o.Text = append(o.Text, line)
		// 	}

		// 	i++
		// 	if i >= len(lines) {
		// 		break
		// 	}

		// 	textLine++
		// }

		// if len(o.Text) > 0 {
		// 	fmt.Println(o.Text)
		// 	fmt.Println(outSeq)
		// 	res.Captions = append(res.Captions, o)
		// 	outSeq++
		// }

	}
	return
}

// AsSRT renders the sub in .srt format
// func (subtitle *Subtitle) AsLRC() (res string) {
// 	for _, sub := range subtitle.Captions {
// 		// res += sub.AsLRC()
// 	}
// 	return
// }

// // TimeSRT renders a timestamp for use in .srt
// func TimeLRC(t time.Time) string {
// 	res := t.Format("15:04:05.000")
// 	return strings.Replace(res, ".", ",", 1)
// }

// parseSrtTime parses a srt subtitle time (duration since start of film)
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
	h, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Now(), err
	}
	m, err := strconv.Atoi(matches[2])
	if err != nil {
		return time.Now(), err
	}
	s, err := strconv.Atoi(matches[3])
	if err != nil {
		return time.Now(), err
	}
	ms, err := strconv.Atoi(matches[4])
	if err != nil {
		return time.Now(), err
	}

	return makeTime(h, m, s, ms), nil
}

// makeTime is a helper to create a time duration
func makeTime(h int, m int, s int, ms int) time.Time {
	return time.Date(0, 1, 1, h, m, s, ms*1000*1000, time.UTC)
}
