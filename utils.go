// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/tramhao/id3v2"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/lyric"
)

// logError logs the error message.
func logError(err error) {
	log.Println("[ERROR]", tracerr.Sprint(err))
}

func logDebug(msg string) {
	log.Println("[DEBUG]", msg)
}

// die logs the error message and call os.Exit(1)
// prefer this instead of panic
func die(err error) {
	logError(err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Formats duration to my desired output mm:ss
func fmtDuration(input time.Duration) string {

	val := input.Round(time.Second).String()

	if !strings.Contains(val, "m") {
		val = "0m" + val
	}
	val = strings.ReplaceAll(val, "h", ":")
	val = strings.ReplaceAll(val, "m", ":")
	val = strings.ReplaceAll(val, "s", "")
	var result []string

	for _, v := range strings.Split(val, ":") {

		if len(v) < 2 {
			result = append(result, "0"+v)
		} else {
			result = append(result, v)
		}

	}

	return strings.Join(result, ":")
}

// fmtDurationH returns the formatted duration `x hr x min`
func fmtDurationH(input time.Duration) string {

	re := regexp.MustCompile(`\d+s`)
	val := input.Round(time.Second).String()

	// remove seconds
	result := re.ReplaceAllString(val, "")
	result = strings.Replace(result, "h", " hr ", 1)
	result = strings.Replace(result, "m", " min", 1)

	if result == "" {
		return "0 hr 0 min"
	}

	return result
}

// Expands relative path to absolute path and tilde to /home/(user)
func expandFilePath(path string) string {
	p := expandTilde(path)

	if filepath.IsAbs(p) {
		return p
	}

	p, err := filepath.Abs(p)
	if err != nil {
		die(err)
	}

	return p
}

// Expands tilde alias to /home/user
func expandTilde(_path string) string {
	if !strings.HasPrefix(_path, "~") {
		return _path
	}

	home, err := os.UserHomeDir()

	if err != nil {
		die(err)
	}

	return path.Join(home, strings.TrimPrefix(_path, "~"))
}

// Detects the filetype of file
func getFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	contentType := http.DetectContentType(buffer)

	return strings.SplitAfter(contentType, "/")[1], nil
}

// Gets the file name by removing extension and path
func getName(fn string) string {
	return strings.TrimSuffix(path.Base(fn), ".mp3")
}

// This just parsing the output from the ytdl to get the audio path
// This is used because we need to get the song name
// example ~/path/to/song/song.mp3
func extractFilePath(output []byte, dir string) string {

	regexSearch := fmt.Sprintf(`\[ffmpeg\] Destination: %s\/.*.mp3`,
		escapeBackSlash(dir))

	parseAudioPathOnly := regexp.MustCompile(`\/.*mp3$`)

	re := regexp.MustCompile(regexSearch)

	return string(parseAudioPathOnly.Find(re.Find(output)))

}

func escapeBackSlash(input string) string {
	return strings.ReplaceAll(input, "/", `\/`)
}

// progresStr creates a simple progress bar
// example: =====-----
func progresStr(progress, maxProgress, maxLength int,
	fill, empty string) string {

	currLength := maxLength * progress / maxProgress

	return fmt.Sprintf("%s%s",
		strings.Repeat(fill, currLength),
		strings.Repeat(empty, maxLength-currLength),
	)
}

// padHex pad the neccessary 0 to create six hex digit
func padHex(r, g, b int32) string {

	var result strings.Builder

	for _, v := range []int32{r, g, b} {
		hex := fmt.Sprintf("%x", v)

		if len(hex) == 1 {
			result.WriteString(fmt.Sprintf("0%s", hex))
		} else {
			result.WriteString(hex)
		}
	}

	return result.String()
}

func validHexColor(color string) bool {
	reg := regexp.MustCompile(`^#([A-Fa-f0-9]{6})$`)
	return reg.MatchString(color)
}

func contains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if needle == i {
			return true
		}
	}
	return false
}

// appendFile appends to a file, create the file if not exists
func appendFile(path string, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if err != os.ErrNotExist {
			return tracerr.Wrap(err)
		}
		// create the neccessary parent directory
		err = os.MkdirAll(filepath.Dir(expandFilePath(path)), os.ModePerm)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func shell(input string) (string, error) {

	args := strings.Split(input, " ")
	for i, arg := range args {
		args[i] = strings.Trim(arg, " ")
	}

	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	if stderr.Len() != 0 {
		return "", errors.New(stderr.String())
	}

	return stdout.String(), nil
}

func embedLyric(songPath string, lyricTobeWritten *lyric.Lyric, isDelete bool) (err error) {

	var tag *id3v2.Tag
	tag, err = id3v2.Open(songPath, id3v2.Options{Parse: true})
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer tag.Close()
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	tag.DeleteFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	// We delete the lyric frame with same language by delete all and add others back
	for _, f := range usltFrames {
		uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			die(errors.New("uslt error"))
		}
		if uslf.ContentDescriptor == lyricTobeWritten.LangExt {
			continue
		}
		tag.AddUnsynchronisedLyricsFrame(uslf)
	}
	syltFrames := tag.GetFrames(tag.CommonID("Synchronised lyrics/text"))
	tag.DeleteFrames(tag.CommonID("Synchronised lyrics/text"))
	for _, f := range syltFrames {
		sylf, ok := f.(id3v2.SynchronisedLyricsFrame)
		if !ok {
			die(errors.New("sylt error"))
		}
		if strings.Contains(sylf.ContentDescriptor, lyricTobeWritten.LangExt) {
			continue
		}
		tag.AddSynchronisedLyricsFrame(sylf)
	}

	if !isDelete {
		tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: lyricTobeWritten.LangExt,
			Lyrics:            lyricTobeWritten.AsLRC(),
		})
		var lyric lyric.Lyric
		err := lyric.NewFromLRC(lyricTobeWritten.AsLRC())
		if err != nil {
			return tracerr.Wrap(err)
		}
		tag.AddSynchronisedLyricsFrame(id3v2.SynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			TimestampFormat:   2,
			ContentType:       1,
			ContentDescriptor: lyricTobeWritten.LangExt,
			SynchronizedTexts: lyric.SyncedCaptions,
		})

	}

	err = tag.Save()
	if err != nil {
		return tracerr.Wrap(err)
	}

	return err

}

func embedLength(songPath string) (time.Duration, error) {
	tag, err := id3v2.Open(songPath, id3v2.Options{Parse: true})
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	defer tag.Close()

	var lengthSongTimeDuration time.Duration
	lengthSongTimeDuration, err = getLength(songPath)
	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	lengthSongString := strconv.FormatInt(lengthSongTimeDuration.Milliseconds(), 10)
	lengthFrame := id3v2.UserDefinedTextFrame{
		Encoding:    id3v2.EncodingUTF8,
		Description: "TLEN",
		Value:       lengthSongString,
	}
	tag.AddUserDefinedTextFrame(lengthFrame)

	err = tag.Save()
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	return lengthSongTimeDuration, err
}

// getLength return the length of the song in the queue
func getLength(audioPath string) (time.Duration, error) {
	f, err := os.Open(audioPath)

	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	defer streamer.Close()
	return format.SampleRate.D(streamer.Len()), nil
}

func getTagLength(songPath string) (songLength time.Duration, err error) {
	var tag *id3v2.Tag
	tag, err = id3v2.Open(songPath, id3v2.Options{Parse: true})
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	defer tag.Close()
	tlenFrames := tag.GetFrames(tag.CommonID("User defined text information frame"))
	if tlenFrames == nil {
		songLength, err = embedLength(songPath)
		if err != nil {
			return 0, tracerr.Wrap(err)
		}
		return songLength, nil
	}
	for _, tlenFrame := range tlenFrames {
		if tlenFrame.(id3v2.UserDefinedTextFrame).Description == "TLEN" {
			songLengthString := tlenFrame.(id3v2.UserDefinedTextFrame).Value
			songLengthInt64, err := strconv.ParseInt(songLengthString, 10, 64)
			if err != nil {
				return 0, tracerr.Wrap(err)
			}
			songLength = (time.Duration)(songLengthInt64) * time.Millisecond
			break
		}
	}
	if songLength != 0 {
		return songLength, nil
	}
	songLength, err = embedLength(songPath)
	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	return songLength, err
}
