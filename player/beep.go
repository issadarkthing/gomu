// Package player is the place actually play the music
package player

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/ztrue/tracerr"
)

type BeepPlayer struct {
	hasInit   bool
	isRunning bool
	volume    float64

	vol              *effects.Volume
	ctrl             *beep.Ctrl
	format           *beep.Format
	length           time.Duration
	currentSong      Audio
	streamSeekCloser beep.StreamSeekCloser

	songFinish func(Audio)
	songStart  func(Audio)
	songSkip   func(Audio)
	mu         *sync.Mutex
}

// NewBeepPlayer returns new Player instance.
func NewBeepPlayer(volume int) (*BeepPlayer, error) {

	// Read initial volume from config
	initVol := AbsVolume(volume)

	// making sure user does not give invalid volume
	if volume > 100 || volume < 0 {
		initVol = 0
	}

	return &BeepPlayer{
		hasInit:          false,
		isRunning:        false,
		volume:           initVol,
		vol:              &effects.Volume{},
		ctrl:             &beep.Ctrl{},
		format:           &beep.Format{},
		length:           0,
		currentSong:      nil,
		streamSeekCloser: nil,
		songFinish: func(Audio) {
		},
		songStart: func(Audio) {
		},
		songSkip: func(Audio) {
		},
		mu: &sync.Mutex{},
	}, nil
}

// SetSongFinish accepts callback which will be executed when the song finishes.
func (p *BeepPlayer) SetSongFinish(f func(Audio)) {
	p.songFinish = f
}

// SetSongStart accepts callback which will be executed when the song starts.
func (p *BeepPlayer) SetSongStart(f func(Audio)) {
	p.songStart = f
}

// SetSongSkip accepts callback which will be executed when the song is skipped.
func (p *BeepPlayer) SetSongSkip(f func(Audio)) {
	p.songSkip = f
}

// executes songFinish callback.
func (p *BeepPlayer) execSongFinish(a Audio) {
	if p.songFinish != nil {
		p.songFinish(a)
	}
}

// executes songStart callback.
func (p *BeepPlayer) execSongStart(a Audio) {
	if p.songStart != nil {
		p.songStart(a)
	}
}

// executes songFinish callback.
func (p *BeepPlayer) execSongSkip(a Audio) {
	if p.songSkip != nil {
		p.songSkip(a)
	}
}

// Run plays the passed Audio.
func (p *BeepPlayer) Run(currSong Audio) error {

	p.isRunning = true
	p.execSongStart(currSong)

	f, err := os.Open(currSong.Path())
	if err != nil {
		return tracerr.Wrap(err)
	}

	stream, format, err := mp3.Decode(f)
	if err != nil {
		return tracerr.Wrap(err)
	}

	p.mu.Lock()
	p.streamSeekCloser = stream
	p.mu.Unlock()

	// song duration
	p.length = format.SampleRate.D(p.streamSeekCloser.Len())

	sr := beep.SampleRate(48000)
	if !p.hasInit {

		// p.mu.Lock()
		err := speaker.Init(sr, sr.N(time.Second/10))
		// p.mu.Unlock()

		if err != nil {
			return tracerr.Wrap(err)
		}

		p.hasInit = true
	}

	p.currentSong = currSong

	// resample to adapt to sample rate of new songs
	resampled := beep.Resample(4, format.SampleRate, sr, p.streamSeekCloser)

	sstreamer := beep.Seq(resampled, beep.Callback(func() {
		p.isRunning = false
		p.format = nil
		p.streamSeekCloser.Close()
		go p.execSongFinish(currSong)
	}))

	ctrl := &beep.Ctrl{
		Streamer: sstreamer,
		Paused:   false,
	}

	p.mu.Lock()
	p.format = &format
	p.ctrl = ctrl
	p.mu.Unlock()
	resampler := beep.ResampleRatio(4, 1, ctrl)

	volume := &effects.Volume{
		Streamer: resampler,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	// sets the volume of previous player
	volume.Volume += p.volume
	p.vol = volume

	// starts playing the audio
	speaker.Play(p.vol)

	return nil
}

// Pause pauses Player.
func (p *BeepPlayer) Pause() {
	speaker.Lock()
	p.ctrl.Paused = true
	p.isRunning = false
	speaker.Unlock()
}

// Play unpauses Player.
func (p *BeepPlayer) Play() {
	speaker.Lock()
	p.ctrl.Paused = false
	p.isRunning = true
	speaker.Unlock()
}

// SetVolume set volume up and volume down using -0.5 or +0.5.
func (p *BeepPlayer) SetVolume(v float64) float64 {

	// check if no songs playing currently
	if p.vol == nil {
		p.volume += v
		return p.volume
	}

	speaker.Lock()
	p.vol.Volume += v
	p.volume = p.vol.Volume
	speaker.Unlock()
	return p.volume
}

// TogglePause toggles the pause state.
func (p *BeepPlayer) TogglePause() {

	if p.ctrl == nil {
		return
	}

	if p.ctrl.Paused {
		p.Play()
	} else {
		p.Pause()
	}
}

// Skip current song.
func (p *BeepPlayer) Skip() error {

	p.execSongSkip(p.currentSong)

	if p.currentSong == nil {
		return errors.New("currentSong is not set")
	}

	// drain the stream
	speaker.Lock()
	p.ctrl.Streamer = nil
	if err := p.streamSeekCloser.Close(); err != nil {
		return tracerr.Wrap(err)
	}
	p.isRunning = false
	p.format = nil
	speaker.Unlock()

	p.execSongFinish(p.currentSong)
	return nil
}

// GetPosition returns the current position of audio file.
func (p *BeepPlayer) GetPosition() time.Duration {

	p.mu.Lock()
	speaker.Lock()
	defer speaker.Unlock()
	defer p.mu.Unlock()
	if p.format == nil || p.streamSeekCloser == nil {
		return 1
	}

	return p.format.SampleRate.D(p.streamSeekCloser.Position())
}

// Seek is the function to move forward and rewind
func (p *BeepPlayer) Seek(pos int) error {
	p.mu.Lock()
	speaker.Lock()
	defer speaker.Unlock()
	defer p.mu.Unlock()
	err := p.streamSeekCloser.Seek(pos * int(p.format.SampleRate))
	return err
}

// IsPaused is used to distinguish the player between pause and stop
func (p *BeepPlayer) IsPaused() bool {
	p.mu.Lock()
	speaker.Lock()
	defer speaker.Unlock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return false
	}

	return p.ctrl.Paused
}

// GetVolume returns current volume.
func (p *BeepPlayer) GetVolume() float64 {
	return p.volume
}

// GetCurrentSong returns current song.
func (p *BeepPlayer) GetCurrentSong() Audio {
	return p.currentSong
}

// HasInit checks if the speaker has been initialized or not. Speaker
// initialization will only happen once.
func (p *BeepPlayer) HasInit() bool {
	return p.hasInit
}

// IsRunning returns true if Player is running an audio.
func (p *BeepPlayer) IsRunning() bool {
	return p.isRunning
}

// VolToHuman converts float64 volume that is used by audio library to human
// readable form (0 - 100)
func (p *BeepPlayer) VolToHuman(volume float64) int {
	return int(volume*10) + 100
}

// AbsVolume converts human readable form volume (0 - 100) to float64 volume
// that is used by the audio library
func AbsVolume(volume int) float64 {
	return (float64(volume) - 100) / 10
}

// Stop is the function which stop the player
func (p *BeepPlayer) Stop() error {
	return nil
}

// UpdateDB is just empty for beep
func (p *BeepPlayer) UpdateDB() error {
	return nil
}
