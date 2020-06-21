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

type handler func(list *tview.List, tree *tview.TreeView, playingBar *tview.Box) 

type Player struct {
	queue     []string
	IsRunning bool
	hasInit   bool
	current   string
	format    *beep.Format
	// to control the _volume internally
	_volume     *effects.Volume
	volume      float64
	position    time.Duration
	length      time.Duration
	currentSong Song

	// to access sections
	list       *tview.List
	tree       *tview.TreeView
	playingBar *Progress
	app        *tview.Application

	afterPlayHandler	handler
	beforePlayHandler   handler
}

func (p *Player) Push(song string) {
	p.queue = append(p.queue, song)
}

func (p *Player) Pop() (string, error) {

	if len(p.queue) == 0 {
		return "", errors.New("Empty list")
	}
	a := p.queue[0]
	p.queue = p.queue[1:]
	p.current = a

	return a, nil
}

func (p *Player) Run() {

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

	done := make(chan bool)

	sstreamer := beep.Seq(streamer, beep.Callback(func() {
		done <- true
	}))

	ctrl := &beep.Ctrl{Streamer: sstreamer, Paused: false}

	volume := &effects.Volume{
		Streamer: ctrl,
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

	go func () {

		i := 0

		for {
			i++
			p.playingBar.progress <- 1

			if i > p.playingBar.full {
				break
			}
			
			time.Sleep(time.Second)
		}

	}()

	for {
		select {
		case <-done:
			close(done)
			p.playingBar._progress = 0
			p.position = 0
			p.current = ""
			if len(p.queue) != 0 {
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


}

func (p *Player) Pause() {
	speaker.Lock()
	p._volume.Streamer.(*beep.Ctrl).Paused = true
	p.IsRunning = false
	speaker.Unlock()
}

func (p *Player) Play() {
	speaker.Lock()
	p._volume.Streamer.(*beep.Ctrl).Paused = false
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

	if p._volume.Streamer.(*beep.Ctrl).Paused {
		p.Play()
	} else {
		p.Pause()
	}
}

func (p *Player) BeforePlayHook(handler func(
	list *tview.List,
	tree *tview.TreeView,
	playingBar *tview.Box),
) {
	p.beforePlayHandler = handler
}

func (p *Player) AfterPlayHook(handler func(
	list *tview.List,
	tree *tview.TreeView,
	playingBar *tview.Box),
) {
	p.afterPlayHandler = handler
}
