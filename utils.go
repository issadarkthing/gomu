// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

func log(text string) {

	f, err := os.OpenFile("message.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	if _, err := f.Write([]byte(text + "\n")); err != nil {
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

}

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

func expandTilde(_path string) string {

	if !strings.HasPrefix(_path, "~") {
		return _path
	}

	home, err := os.UserHomeDir()

	if err != nil {
		log(err.Error())
	}

	return path.Join(home, strings.TrimPrefix(_path, "~"))

}

// gets the length of the song in the queue
func GetLength(audioPath string) (time.Duration, error) {

	f, err := os.Open(audioPath)

	defer f.Close()

	if err != nil {
		return 0, err
	}

	streamer, format, err := mp3.Decode(f)

	defer streamer.Close()

	if err != nil {
		return 0, err
	}

	return format.SampleRate.D(streamer.Len()), nil
}

// detects the filetype of file
func GetFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return strings.SplitAfter(contentType, "/")[1], nil
}

// gets the file name by removing extension and path
func GetName(fn string) string {
	return strings.TrimSuffix(path.Base(fn), path.Ext(fn))
}

// download audio from youtube audio and adds the song to the selected playlist
func Ytdl(url string, selPlaylist *tview.TreeNode) {

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.Name

	timeoutPopup(" Ytdl ", "Downloading", time.Second*5)

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%s/%%(artist)s - %%(track)s.%%(ext)s", 
		dir, 
		selPlaylistName)

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {

		err := cmd.Run()
		if err != nil {
			timeoutPopup(" Error ", "An error occured when downloading", time.Second*5)
			return
		}

		if cmd.Stderr != nil {
			timeoutPopup(" Error ", "An error occured when downloading", time.Second*5)
			return
		}

		playlistPath := fmt.Sprintf("%s/%s", expandTilde(dir), selPlaylistName)

		downloadedAudioPath := downloadedFilePath(
			stdout.Bytes(), playlistPath)

		f, err := os.Open(downloadedAudioPath)

		if err != nil {
			log(err.Error())
		}

		defer f.Close()

		node := tview.NewTreeNode(path.Base(f.Name()))

		audioFile := &AudioFile{
			Name:        f.Name(),
			Path:        downloadedAudioPath,
			IsAudioFile: true,
			Parent:      selPlaylist,
		}

		node.SetReference(audioFile)
		selPlaylist.AddChild(node)
		app.Draw()

		timeoutPopup(
			" Ytdl ",
			fmt.Sprintf("Finished downloading\n%s", 
			path.Base(downloadedAudioPath)), time.Second*5)

	}()

}

// this just parsing the output from the ytdl to get the audio path
// this is used because we need to get the song name
// example ~/path/to/song/song.mp3
func downloadedFilePath(output []byte, dir string) string {

	regexSearch := fmt.Sprintf(`\[ffmpeg\] Destination: %s\/.*.mp3`,
		escapeBackSlash(dir))

	parseAudioPathOnly := regexp.MustCompile(`\/.*mp3$`)

	re := regexp.MustCompile(regexSearch)

	return string(parseAudioPathOnly.Find(re.Find(output)))

}

func escapeBackSlash(input string) string {
	return strings.ReplaceAll(input, "/", `\/`)
}
