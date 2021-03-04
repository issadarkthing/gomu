package player

import (
	"os"
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
	p.songFinish = f
}

// SetSongStart accepts callback which will be executed when the song starts.
func (p *Player) SetSongStart(f func(Audio)) {
	p.songStart = f
}

// SetSongSkip accepts callback which will be executed when the song is skipped.
func (p *Player) SetSongSkip(f func(Audio)) {
	p.songSkip = f
}

// executes songFinish callback.
func (p *Player) execSongFinish(a Audio) {
	if p.songFinish != nil {
		p.songFinish(a)
	}
}

// executes songStart callback.
func (p *Player) execSongStart(a Audio) {
	if p.songStart != nil {
		p.songStart(a)
	}
}

// executes songFinish callback.
func (p *Player) execSongSkip(a Audio) {
	if p.songSkip != nil {
		p.songSkip(a)
	}
}

// Run plays the passed Audio.
func (p *Player) Run(currSong Audio) error {

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

	p.streamSeekCloser = stream
	p.format = &format

	// song duration
	p.length = p.format.SampleRate.D(p.streamSeekCloser.Len())

	sr := beep.SampleRate(48000)
	if !p.hasInit {

		err := speaker.Init(sr, sr.N(time.Second/10))

		if err != nil {
			return tracerr.Wrap(err)
		}

		p.hasInit = true
	}

	p.currentSong = currSong

	// resample to adapt to sample rate of new songs
	resampled := beep.Resample(4, p.format.SampleRate, sr, p.streamSeekCloser)

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

	p.ctrl = ctrl
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
func (p *Player) Pause() {
	speaker.Lock()
	p.ctrl.Paused = true
	p.isRunning = false
	speaker.Unlock()
}

// Play unpauses Player.
func (p *Player) Play() {
	speaker.Lock()
	p.ctrl.Paused = false
	p.isRunning = true
	speaker.Unlock()
}

// Volume up and volume down using -0.5 or +0.5.
func (p *Player) SetVolume(v float64) float64 {

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

// Toggles the pause state.
func (p *Player) TogglePause() {

	if p.ctrl == nil {
		return
	}

	if p.ctrl.Paused {
		p.Play()
	} else {
		p.Pause()
	}
}

// Skips current song.
func (p *Player) Skip() {

	p.execSongSkip(p.currentSong)

	if p.currentSong == nil {
		return
	}

	// drain the stream
	p.ctrl.Streamer = nil

	p.streamSeekCloser.Close()
	p.isRunning = false
	p.format = nil
	p.execSongFinish(p.currentSong)
}

// GetPosition returns the current position of audio file.
func (p *Player) GetPosition() time.Duration {

	if p.format == nil || p.streamSeekCloser == nil {
		return 1
	}

	return p.format.SampleRate.D(p.streamSeekCloser.Position())
}

// seek is the function to move forward and rewind
func (p *Player) Seek(pos int) error {
	speaker.Lock()
	defer speaker.Unlock()
	err := p.streamSeekCloser.Seek(pos * int(p.format.SampleRate))
	return err
}

// IsPaused is used to distinguish the player between pause and stop
func (p *Player) IsPaused() bool {
	if p.ctrl == nil {
		return false
	}

	return p.ctrl.Paused
}

// GetVolume returns current volume.
func (p *Player) GetVolume() float64 {
	return p.volume
}

// GetCurrentSong returns current song.
func (p *Player) GetCurrentSong() Audio {
	return p.currentSong
}

// HasInit checks if the speaker has been initialized or not. Speaker
// initialization will only happen once.
func (p *Player) HasInit() bool {
	return p.hasInit
}

// IsRunning returns true if Player is running an audio.
func (p *Player) IsRunning() bool {
	return p.isRunning
}

// Gets the length of the song in the queue
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
