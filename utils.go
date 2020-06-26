// Copyright (C) 2020  Raziman

package main

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/faiface/beep/mp3"
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
