// Copyright (C) 2020  Raziman

package main

import (
	"os"
	"path"
	"strings"
	"time"
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
			result = append(result, "0" + v)
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
