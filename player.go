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
	hasInit   bool
	format    *beep.Format
	isLoop    bool
	isRunning bool
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

func newPlayer() *Player {
	// Read initial volume from config
	initVol := absVolume(viper.GetInt("volume"))

	// making sure user does not give invalid volume
	if initVol > 100 {
		initVol = 100
	} else if initVol < 1 {
		initVol = 0
	}

	return &Player{volume: initVol}
}

func (p *Player) run(currSong *AudioFile) error {

	p.isSkipped = make(chan bool, 1)
	f, err := os.Open(currSong.path)

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
		currSong.name, fmtDuration(p.length))

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
	p.isRunning = true

	if p.isLoop {
		gomu.queue.enqueue(currSong)
		gomu.app.Draw()
	}

	gomu.playingBar.newProgress(currSong.name, int(p.length.Seconds()), 100)

	go func() {
		if err := gomu.playingBar.run(); err != nil {
			logError(err)
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
			p.isRunning = false
			p.format = nil
			gomu.playingBar.stop()

			nextSong, err := gomu.queue.dequeue()
			gomu.app.Draw()

			if err != nil {
				break next
			}

			go func() {
				if err := p.run(nextSong); err != nil {
					logError(err)
				}
			}()

			break next

		case <-time.After(time.Second):
			// stop progress bar from progressing when paused
			if !p.isRunning {
				continue
			}

			i++
			gomu.playingBar.progress <- 1

			speaker.Lock()
			p.position = position()
			speaker.Unlock()

			if i > gomu.playingBar.full {
				break next
			}

		}

	}

	return nil
}

func (p *Player) pause() {
	speaker.Lock()
	p.ctrl.Paused = true
	p.isRunning = false
	speaker.Unlock()
}

func (p *Player) play() {
	speaker.Lock()
	p.ctrl.Paused = false
	p.isRunning = true
	speaker.Unlock()
}

// volume up and volume down using -0.5 or +0.5
func (p *Player) setVolume(v float64) float64 {

	// check if no songs playing currently
	if p._volume == nil {
		p.volume += v
		return p.volume
	}

	defer func() {
		// saves the volume
		volume := volToHuman(p.volume)
		viper.Set("volume", volume)
	}()

	speaker.Lock()
	p._volume.Volume += v
	p.volume = p._volume.Volume
	speaker.Unlock()
	return p.volume
}

func (p *Player) togglePause() {

	if p.ctrl == nil {
		return
	}

	if p.ctrl.Paused {
		p.play()
	} else {
		p.pause()
	}
}

// skips current song
func (p *Player) skip() {

	if gomu.queue.GetItemCount() < 1 {
		return
	}

	p.ctrl.Streamer = nil
	p.done <- true
}

// Toggles the queue to loop
// dequeued item will be enqueued back
// function returns loop state
func (p *Player) toggleLoop() bool {
	p.isLoop = !p.isLoop
	return p.isLoop
}

// Gets the length of the song in the queue
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

// volToHuman converts float64 volume that is used by audio library to human
// readable form (0 - 100)
func volToHuman(volume float64) int {
	return int(volume*10) + 100
}

// absVolume converts human readable form volume (0 - 100) to float64 volume
// that is used by the audio library
func absVolume(volume int) float64 {
	return (float64(volume) - 100) / 10
}
