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
)

type Player struct {
	IsRunning bool
	hasInit   bool
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
	currentSong *AudioFile
}

func NewPlayer() *Player {

	// Read initial volume from config
	var initVol float64 = (viper.GetFloat64("volume") - 50.0) / 10.0

	return &Player{volume: initVol}
}

func (p *Player) Run(currSong *AudioFile) {

	p.isSkipped = make(chan bool, 1)

	f, err := os.Open(currSong.Path)

	if err != nil {
		appLog(err)
	}

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	// song duration
	p.length = format.SampleRate.D(streamer.Len())

	if !p.hasInit {
		err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

		if err != nil {
			appLog(err)
		}

		p.hasInit = true
	}

	p.format = &format

	if err != nil {
		appLog(err)
	}

	defer streamer.Close()

	if err != nil {
		appLog(err)
	}

	p.currentSong = currSong

	popupMessage := fmt.Sprintf("%s\n\n[ %s ]", currSong.Name, fmtDuration(p.length))

	timedPopup(" Current Song ", popupMessage, getPopupTimeout())

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

	gomu.PlayingBar.NewProgress(currSong.Name, int(p.length.Seconds()), 100)
	gomu.PlayingBar.Run()

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

			go p.Run(nextSong)

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

	if gomu.Queue.GetItemCount() > 0 {
		p.ctrl.Streamer = nil
		p.done <- true
	}
}

// gets the length of the song in the queue
func GetLength(audioPath string) (time.Duration, error) {

	f, err := os.Open(audioPath)

	if err != nil {
		return 0, err
	}

	defer f.Close()

	streamer, format, err := mp3.Decode(f)

	if err != nil {
		return 0, err
	}

	defer streamer.Close()

	return format.SampleRate.D(streamer.Len()), nil
}
