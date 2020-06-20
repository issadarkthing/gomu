package main

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type Song struct {
	name     string
	path     string
	position string
	format   string
}

type Player struct {
	list        []string
	IsRunning   bool
	hasInit     bool
	ctrl        *beep.Ctrl
	current     string
	format      *beep.Format
	position    string
	currentSong Song
}

func Init(paths []string) (*Player, error) {

	p := &Player{list: paths, IsRunning: false, hasInit: false}

	if len(paths) == 0 {
		return nil, errors.New("Cannot play with empty list")
	}

	return p, nil

}

func (p *Player) Push(song string) {
	p.list = append(p.list, song)
}

func (p *Player) Pop() (string, error) {

	if len(p.list) == 0 {
		return "", errors.New("Empty list")
	}
	a := p.list[0]
	p.list = p.list[1:]
	p.current = a

	return a, nil
}

func (p *Player) Run() error {

	first, err := p.Pop()

	if err != nil {
		p.IsRunning = false
		return err
	}

	f, err := os.Open(first)

	fformat, err := GetFileContentType(f)

	if err != nil {
		return err
	}

	song := &Song{name: GetName(f.Name()), path: first, format: fformat}
	p.currentSong = *song

	if err != nil {
		return err
	}

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	p.format = &format

	if err != nil {
		return err
	}

	defer streamer.Close()

	if !p.hasInit {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		p.hasInit = true
	}

	done := make(chan bool)

	sstreamer := beep.Seq(streamer, beep.Callback(func() {
		done <- true
		close(done)
	}))

	p.ctrl = &beep.Ctrl{Streamer: sstreamer, Paused: false}

	speaker.Play(p.ctrl)
	position := func() string {
		return format.SampleRate.D(streamer.Position()).Round(time.Second).String()
	}

	p.position = position()

	p.IsRunning = true

	for {
		select {
		case <-done:
			p.position = ""
			p.current = ""
			if len(p.list) != 0 {
				go p.Run()
			} else {
				p.IsRunning = false
				p.format = nil
			}
			goto next
		case <-time.After(time.Second):
			speaker.Lock()
			p.position = position()
			speaker.Unlock()
		}
	}

next:

	return nil

}

func (p *Player) Pause() {
	speaker.Lock()
	p.ctrl.Paused = true
	p.IsRunning = false
	speaker.Unlock()
}

func (p *Player) Play() {
	speaker.Lock()
	p.ctrl.Paused = false
	p.IsRunning = true
	speaker.Unlock()
}

func (p *Player) CurrentSong() Song {
	return p.currentSong
}


func GetFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return strings.SplitAfter(contentType, "/")[1], nil
}

func GetName(fn string) string {
      return strings.TrimSuffix(fn, path.Ext(fn))
}
