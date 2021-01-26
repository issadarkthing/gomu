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

var (
  s   beep.StreamSeekCloser
  ft  beep.Format
	// is used to send progress
  i   int
)

func newPlayer() *Player {

	volume := viper.GetInt("general.volume")
	// Read initial volume from config
	initVol := absVolume(volume)

	// making sure user does not give invalid volume
	if volume > 100 || volume < 0 {
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

	s, ft, err = mp3.Decode(f)

	if err != nil {
		return tracerr.Wrap(err)
	}

	defer s.Close()
  
	// song duration
	p.length = ft.SampleRate.D(s.Len())

	if !p.hasInit {

		err := speaker.
			Init(ft.SampleRate, ft.SampleRate.N(time.Second))

		if err != nil {
			return tracerr.Wrap(err)
		}

		p.hasInit = true
	}

	p.format = &ft
	p.currentSong = currSong

	popupMessage := fmt.Sprintf("%s\n\n[ %s ]",
		currSong.name, fmtDuration(p.length))

	defaultTimedPopup(" Current Song ", popupMessage)

	done := make(chan bool, 1)
	p.done = done

	sstreamer := beep.Seq(s, beep.Callback(func() {
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

	p.position = position()
	p.isRunning = true

	if p.isLoop {
		gomu.queue.enqueue(currSong)
		gomu.app.Draw()
	}

	gomu.playingBar.newProgress(currSong.name, int(p.length.Seconds()))

	go func() {
		if err := gomu.playingBar.run(); err != nil {
			logError(err)
		}
	}()

	// is used to send progress
	i = 0

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

	defer func() {
		// saves the volume
		volume := volToHuman(p.volume)
		viper.Set("general.volume", volume)
	}()

	// check if no songs playing currently
	if p._volume == nil {
		p.volume += v
		return p.volume
	}

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

func position() time.Duration {
		return ft.SampleRate.D(s.Position())
}

//seek is the function to move forward and rewind
func (p *Player) seek(pos int) error {
  speaker.Lock()
  err := s.Seek(pos * int(ft.SampleRate))
  i = pos
  speaker.Unlock()
  return err
}

