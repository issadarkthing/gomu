// Package player is the place actually play the music
package player

import (
	"os"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/ztrue/tracerr"
)

type Audio interface {
	Name() string
	Path() string
}

type Player struct {
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
	mu         sync.Mutex
}

// New returns new Player instance.
func New(volume int) *Player {

	// Read initial volume from config
	initVol := AbsVolume(volume)

	// making sure user does not give invalid volume
	if volume > 100 || volume < 0 {
		initVol = 0
	}

	return &Player{volume: initVol}
}

// SetSongFinish accepts callback which will be executed when the song finishes.
func (p *Player) SetSongFinish(f func(Audio)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.songFinish = f
}

// SetSongStart accepts callback which will be executed when the song starts.
func (p *Player) SetSongStart(f func(Audio)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.songStart = f
}

// SetSongSkip accepts callback which will be executed when the song is skipped.
func (p *Player) SetSongSkip(f func(Audio)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.songSkip = f
}

// executes songFinish callback.
func (p *Player) execSongFinish(a Audio) {
	p.mu.Lock()
	f := p.songFinish
	p.mu.Unlock()
	if f != nil {
		f(a)
	}
}

// executes songStart callback.
func (p *Player) execSongStart(a Audio) {
	p.mu.Lock()
	f := p.songStart
	p.mu.Unlock()
	if f != nil {
		f(a)
	}
}

// executes songFinish callback.
func (p *Player) execSongSkip(a Audio) {
	p.mu.Lock()
	f := p.songSkip
	p.mu.Unlock()
	if f != nil {
		f(a)
	}
}

// Run plays the passed Audio.
func (p *Player) Run(currSong Audio) error {
	p.mu.Lock()
	p.isRunning = true
	p.currentSong = currSong
	p.mu.Unlock()

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
	p.length = format.SampleRate.D(stream.Len())
	hasInit := p.hasInit
	p.mu.Unlock()

	sr := beep.SampleRate(48000)
	if !hasInit {
		err := speaker.Init(sr, sr.N(time.Second/10))
		if err != nil {
			return tracerr.Wrap(err)
		}
		p.mu.Lock()
		p.hasInit = true
		p.mu.Unlock()
	}

	// resample to adapt to sample rate of new songs
	resampled := beep.Resample(4, format.SampleRate, sr, stream)

	sstreamer := beep.Seq(resampled, beep.Callback(func() {
		go func() {
			p.mu.Lock()
			p.isRunning = false
			p.format = nil
			if p.streamSeekCloser != nil {
				p.streamSeekCloser.Close()
			}
			p.mu.Unlock()
			p.execSongFinish(currSong)
		}()
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

	p.mu.Lock()
	volume.Volume += p.volume
	p.vol = volume
	p.mu.Unlock()

	// starts playing the audio
	speaker.Play(volume)

	return nil
}

// Pause pauses Player.
func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	speaker.Lock()
	if p.ctrl != nil {
		p.ctrl.Paused = true
	}
	p.isRunning = false
	speaker.Unlock()
}

// Play unpauses Player.
func (p *Player) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()
	speaker.Lock()
	if p.ctrl != nil {
		p.ctrl.Paused = false
	}
	p.isRunning = true
	speaker.Unlock()
}

// SetVolume set volume up and volume down using -0.5 or +0.5.
func (p *Player) SetVolume(v float64) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

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
func (p *Player) TogglePause() {
	p.mu.Lock()
	ctrl := p.ctrl
	p.mu.Unlock()

	if ctrl == nil {
		return
	}

	speaker.Lock()
	paused := ctrl.Paused
	speaker.Unlock()

	if paused {
		p.Play()
	} else {
		p.Pause()
	}
}

// Skip current song.
func (p *Player) Skip() {
	p.mu.Lock()
	currentSong := p.currentSong
	p.mu.Unlock()

	if currentSong == nil {
		return
	}

	p.execSongSkip(currentSong)

	p.mu.Lock()
	speaker.Lock()
	if p.ctrl != nil {
		p.ctrl.Streamer = nil
	}
	if p.streamSeekCloser != nil {
		p.streamSeekCloser.Close()
	}
	p.isRunning = false
	p.format = nil
	speaker.Unlock()
	p.mu.Unlock()

	p.execSongFinish(currentSong)
}

// GetPosition returns the current position of audio file.
func (p *Player) GetPosition() time.Duration {
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
func (p *Player) Seek(pos int) error {
	p.mu.Lock()
	speaker.Lock()
	defer speaker.Unlock()
	defer p.mu.Unlock()
	if p.streamSeekCloser == nil {
		return nil
	}
	err := p.streamSeekCloser.Seek(pos * int(p.format.SampleRate))
	return err
}

// IsPaused is used to distinguish the player between pause and stop
func (p *Player) IsPaused() bool {
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
func (p *Player) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// GetCurrentSong returns current song.
func (p *Player) GetCurrentSong() Audio {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSong
}

// HasInit checks if the speaker has been initialized or not. Speaker
// initialization will only happen once.
func (p *Player) HasInit() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hasInit
}

// IsRunning returns true if Player is running an audio.
func (p *Player) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isRunning
}

// GetLength return the length of the song in the queue
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

// VolToHuman converts float64 volume that is used by audio library to human
// readable form (0 - 100)
func VolToHuman(volume float64) int {
	return int(volume*10) + 100
}

// AbsVolume converts human readable form volume (0 - 100) to float64 volume
// that is used by the audio library
func AbsVolume(volume int) float64 {
	return (float64(volume) - 100) / 10
}
