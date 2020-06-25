// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/rivo/tview"
)


type Song struct {
	name     string
	path     string
	position string
}

type Player struct {
	queue     []string
	IsRunning bool
	hasInit   bool
	current   string
	format    *beep.Format

	isSkipped chan bool
	done      chan bool

	// to control the _volume internally
	_volume     *effects.Volume
	ctrl        *beep.Ctrl
	volume      float64
	resampler   *beep.Resampler
	position    time.Duration
	length      time.Duration
	currentSong Song

	// to access sections
	list       *tview.List
	tree       *tview.TreeView
	playingBar *Progress
	app        *tview.Application
}

// add new song to the queue
func (p *Player) Push(song string) {
	p.queue = append(p.queue, song)
}

// remove first song from the queue
func (p *Player) Pop() (string, error) {

	if len(p.queue) == 0 {
		return "", errors.New("Empty list")
	}
	a := p.queue[0]
	p.queue = p.queue[1:]
	p.current = a

	return a, nil
}

// remove song from the queue
func (p *Player) Remove(index int) (string, error) {

	if index > len(p.queue)-1 {
		return "", errors.New("Index out of range")
	}

	removed := p.queue[index]

	var rest []string

	// check if given index is the last element
	if index == len(p.queue)-1 {
		rest = []string{}
	} else {
		rest = p.queue[index+1:]
	}

	p.queue = append(p.queue[:index], rest...)

	return removed, nil
}

func (p *Player) Run() {

	p.isSkipped = make(chan bool, 1)
	first, err := p.Pop()

	// removes playing song from the queue
	p.list.RemoveItem(0)
	p.app.Draw()

	if err != nil {
		p.IsRunning = false
		log(err.Error())
	}
	f, err := os.Open(first)

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	// song duration
	p.length = format.SampleRate.D(streamer.Len())

	if !p.hasInit {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		p.hasInit = true
	}

	p.format = &format

	if err != nil {
		log(err.Error())
	}

	defer streamer.Close()

	if err != nil {
		log(err.Error())
	}

	song := &Song{name: GetName(f.Name()), path: first}
	p.currentSong = *song

	done := make(chan bool, 1)

	p.done = done

	sstreamer := beep.Seq(streamer, beep.Callback(func() {
		done <- true
	}))

	ctrl := &beep.Ctrl{Streamer: sstreamer, Paused: false}
	p.ctrl = ctrl

	resampler := beep.ResampleRatio(4, 1, ctrl)
	p.resampler = resampler

	volume := &effects.Volume{
		Streamer: resampler,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	// sets the volume of previous player
	volume.Volume += p.volume

	p._volume = volume

	speaker.Play(p._volume)

	position := func() time.Duration {
		return format.SampleRate.D(streamer.Position())
	}

	p.position = position()
	p.IsRunning = true

	p.playingBar.NewProgress(song.name, int(p.length.Seconds()), 100)
	p.playingBar.Run()

	// is used to send progress
	i := 0

next:
	for {

		select {
		case <-done:
			close(done)
			p.position = 0
			p.current = ""
			p.IsRunning = false
			p.format = nil
			p.playingBar.Stop()

			if len(p.queue) != 0 {
				go p.Run()
			}
			break next

		case <-time.After(time.Second):
			// stop progress bar from progressing when paused
			if !p.IsRunning {
				continue
			}

			i++
			p.playingBar.progress <- 1

			speaker.Lock()
			p.position = position()
			speaker.Unlock()

			if i > p.playingBar.full {
				break next
			}

		}

	}


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
	return strings.TrimSuffix(path.Base(fn), path.Ext(fn))
}

// volume up and volume down using -0.5 or +0.5
func (p *Player) Volume(v float64) {
	speaker.Lock()
	p._volume.Volume += v
	p.volume = p._volume.Volume
	speaker.Unlock()
}

func (p *Player) TogglePause() {

	if p.ctrl.Paused {
		p.Play()
	} else {
		p.Pause()
	}
}

// skips current song
func (p *Player) Skip() {
	if len(p.queue) > 0 {
		p.ctrl.Streamer = nil
		p.done <- true
	}
}

// gets the length of the song in the queue
func (p *Player) GetLength(index int) (time.Duration, error) {
	
	if index > len(p.queue)-1 {
		return 0, errors.New("Index out of range")
	}

	f, err := os.Open(p.queue[index])

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
