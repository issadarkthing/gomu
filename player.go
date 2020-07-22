// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

type Player struct {
	IsRunning bool
	hasInit   bool
	format    *beep.Format
	isLoop    bool
	isSkipped chan bool
	done      chan bool

	// to control the _volume internally
	_volume     *effects.Volume
	ctrl        *beep.Ctrl
	volume      float64
	resampler   *beep.Resampler
	position    time.Duration
	length      time.Duration
	currentSong *AudioFile
}

func NewPlayer() *Player {

	// Read initial volume from config
	var initVol float64 = (viper.GetFloat64("volume") - 50.0) / 10.0

	return &Player{volume: initVol}
}

func (p *Player) Run(currSong *AudioFile) error {

	p.isSkipped = make(chan bool, 1)

	f, err := os.Open(currSong.Path)

	if err != nil {
		return tracerr.Wrap(err)
	}

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	if err != nil {
		return tracerr.Wrap(err)
	}

	defer streamer.Close()

	// song duration
	p.length = format.SampleRate.D(streamer.Len())

	if !p.hasInit {

		err := speaker.
			Init(format.SampleRate, format.SampleRate.N(time.Second/10))

		if err != nil {
			return tracerr.Wrap(err)
		}

		p.hasInit = true
	}

	p.format = &format
	p.currentSong = currSong

	popupMessage := fmt.Sprintf("%s\n\n[ %s ]",
		currSong.Name, fmtDuration(p.length))

	timedPopup(" Current Song ", popupMessage, getPopupTimeout(), 0, 0)

	done := make(chan bool, 1)
	p.done = done

	sstreamer := beep.Seq(streamer, beep.Callback(func() {
		done <- true
	}))

	ctrl := &beep.Ctrl{
		Streamer: sstreamer,
		Paused:   false,
	}

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

	if p.isLoop {
		gomu.Queue.Enqueue(currSong)
		gomu.App.Draw()
	}

	gomu.PlayingBar.NewProgress(currSong.Name, int(p.length.Seconds()), 100)

	go func() {
		if err := gomu.PlayingBar.Run(); err != nil {
			LogError(err)
		}
	}()

	// is used to send progress
	i := 0

next:
	for {

		select {
		case <-done:
			close(done)
			p.position = 0
			p.IsRunning = false
			p.format = nil
			gomu.PlayingBar.Stop()

			nextSong, err := gomu.Queue.Dequeue()
			gomu.App.Draw()

			if err != nil {
				break next
			}

			go func() {
				if err := p.Run(nextSong); err != nil {
					LogError(err)
				}
			}()

			break next

		case <-time.After(time.Second):
			// stop progress bar from progressing when paused
			if !p.IsRunning {
				continue
			}

			i++
			gomu.PlayingBar.progress <- 1

			speaker.Lock()
			p.position = position()
			speaker.Unlock()

			if i > gomu.PlayingBar.full {
				break next
			}

		}

	}

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

// volume up and volume down using -0.5 or +0.5
func (p *Player) Volume(v float64) float64 {

	// check if no songs playing currently
	if p._volume == nil {
		p.volume += v
		return p.volume
	}

	defer func() {
		// saves the volume
		volume := int(p.volume*10) + 50
		viper.Set("volume", volume)
	}()

	speaker.Lock()
	p._volume.Volume += v
	p.volume = p._volume.Volume
	speaker.Unlock()
	return p.volume
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

	if gomu.Queue.GetItemCount() < 1 {
		return
	}

	p.ctrl.Streamer = nil
	p.done <- true
}

// Toggles the queue to loop
// dequeued item will be enqueued back
// function returns loop state
func (p *Player) ToggleLoop() bool {
	p.isLoop = !p.isLoop
	return p.isLoop
}

// gets the length of the song in the queue
func GetLength(audioPath string) (time.Duration, error) {

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
