// Package player is the place actually play the music
package player

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/ztrue/tracerr"
)

type MPDPlayer struct {
	hasInit     bool
	isRunning   bool
	vol         int
	volume      float64
	currentSong Audio
	mpdPort     string

	songFinish func(Audio)
	songStart  func(Audio)
	songSkip   func(Audio)
	client     *mpd.Client
	mu         sync.Mutex
}

// NewMPDPlayer returns new Player instance.
func NewMPDPlayer(volume int, mpdPort string) (*MPDPlayer, error) {

	// Read initial volume from config
	initVol := AbsVolume(volume)

	// making sure user does not give invalid volume
	if volume > 100 || volume < 0 {
		initVol = 0
	}

	mpdConn, err := mpd.Dial("tcp", mpdPort)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	if err := mpdConn.Consume(true); err != nil { // Remove song when finished playing
		return nil, tracerr.Wrap(err)
	}
	if err := mpdConn.Clear(); err != nil { // Clear mpd on startup
		return nil, tracerr.Wrap(err)
	}

	log.Println("Successfully connected to MPD")

	if err := mpdConn.SetVolume(volume); err != nil {
		return nil, tracerr.Wrap(err)
	}

	if _, err := mpdConn.Update(""); err != nil {
		return nil, tracerr.Wrap(err)
	}

	return &MPDPlayer{
		hasInit:     true,
		isRunning:   false,
		vol:         volume,
		volume:      initVol,
		currentSong: nil,
		mpdPort:     mpdPort,
		songFinish: func(Audio) {
		},
		songStart: func(Audio) {
		},
		songSkip: func(Audio) {
		},
		client: mpdConn,
		mu:     sync.Mutex{},
	}, nil
}

// SetSongFinish accepts callback which will be executed when the song finishes.
func (p *MPDPlayer) SetSongFinish(f func(Audio)) {
	p.songFinish = f
}

// SetSongStart accepts callback which will be executed when the song starts.
func (p *MPDPlayer) SetSongStart(f func(Audio)) {
	p.songStart = f
}

// SetSongSkip accepts callback which will be executed when the song is skipped.
func (p *MPDPlayer) SetSongSkip(f func(Audio)) {
	p.songSkip = f
}

// executes songFinish callback.
func (p *MPDPlayer) execSongFinish(a Audio) {
	if p.songFinish != nil {
		p.songFinish(a)
	}
}

// executes songStart callback.
func (p *MPDPlayer) execSongStart(a Audio) {
	if p.songStart != nil {
		p.songStart(a)
	}
}

// executes songFinish callback.
func (p *MPDPlayer) execSongSkip(a Audio) {
	if p.songSkip != nil {
		p.songSkip(a)
	}
}

// Run plays the passed Audio.
func (p *MPDPlayer) Run(currSong Audio) (err error) {

	p.isRunning = true
	p.execSongStart(currSong)

	if p.client == nil {
		if err = p.reconnect(); err != nil {
			return tracerr.Wrap(err)
		}
	}
	status, err := p.client.Status()
	if err != nil {
		log.Fatalln(err)
	}

	if status["state"] == "play" {
		if err = p.client.Stop(); err != nil {
			return tracerr.Wrap(err)
		}
		if err = p.client.Clear(); err != nil {
			return tracerr.Wrap(err)
		}
	}
	files, err := p.client.GetFiles()
	if err != nil {
		return tracerr.Wrap(err)
	}
	p.currentSong = currSong

	fileName := p.currentSong.Name()
	for _, f := range files {
		if !strings.Contains(f, fileName) {
			continue
		}
		if err = p.client.Add(f); err != nil {
			return tracerr.Wrap(err)
		}
		break
	}
	if err = p.client.Play(-1); err != nil {
		return tracerr.Wrap(err)
	}

	go func() {
		for {
			if p.client == nil {
				if err = p.reconnect(); err != nil {
					return
				}
			}

			status, err := p.client.Status()
			if err != nil {
				log.Fatalln(err)
			}

			if status["state"] == "stop" {
				p.isRunning = false
				p.execSongFinish(currSong)
				break
			}

			<-time.After(time.Second)
		}
	}()

	return nil
}

// Pause pauses Player.
func (p *MPDPlayer) Pause() {
	p.client.Pause(true)
	p.isRunning = false
}

// Play unpauses Player.
func (p *MPDPlayer) Play() {
	p.client.Pause(false)
	p.isRunning = true
}

// SetVolume set volume up and volume down using -0.5 or +0.5.
func (p *MPDPlayer) SetVolume(v float64) float64 {

	p.volume += v
	p.client.SetVolume(p.VolToHuman(p.volume))
	return p.volume
}

// TogglePause toggles the pause state.
func (p *MPDPlayer) TogglePause() {

	if p.isRunning {
		p.Pause()
		return
	}

	p.Play()

}

// Skip current song.
func (p *MPDPlayer) Skip() {

	p.execSongSkip(p.currentSong)

	if p.currentSong == nil {
		return
	}

	p.isRunning = false

	p.execSongFinish(p.currentSong)
}

// GetPosition returns the current position of audio file.
func (p *MPDPlayer) GetPosition() time.Duration {

	if !p.hasInit {
		return 0
	}

	if !p.isRunning {
		return 0
	}
	if p.client == nil {
		if err := p.reconnect(); err != nil {
			return 0
		}
	}

	status, err := p.client.Status()
	if err != nil {
		return 0
	}

	if status["elapsed"] == "" {
		return 0
	}
	elapsed, err := strconv.ParseFloat(status["elapsed"], 64)
	if err != nil {
		log.Fatalln(err)
	}
	return time.Duration(elapsed) * time.Second
}

// Seek is the function to move forward and rewind
func (p *MPDPlayer) Seek(pos int) error {
	if p.client == nil {
		if err := p.reconnect(); err != nil {
			return tracerr.Wrap(err)
		}
	}

	if err := p.client.SeekCur(time.Duration(pos)*time.Second, false); err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// IsPaused is used to distinguish the player between pause and stop
func (p *MPDPlayer) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !(p.isRunning)
}

// GetVolume returns current volume.
func (p *MPDPlayer) GetVolume() float64 {
	return p.volume
}

// GetCurrentSong returns current song.
func (p *MPDPlayer) GetCurrentSong() Audio {
	return p.currentSong
}

// HasInit checks if the speaker has been initialized or not. Speaker
// initialization will only happen once.
func (p *MPDPlayer) HasInit() bool {
	return p.hasInit
}

// IsRunning returns true if Player is running an audio.
func (p *MPDPlayer) IsRunning() bool {
	return p.isRunning
}

// GetLength return the length of the song in the queue
func (p *MPDPlayer) GetLength(audioPath string) (time.Duration, error) {

	if !p.hasInit {
		return 0, nil
	}
	if p.client == nil {
		if err := p.reconnect(); err != nil {
			return 0, tracerr.Wrap(err)
		}
	}

	status, err := p.client.Status()
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	total, err := strconv.ParseFloat(status["duration"], 64)
	if err != nil {
		log.Fatalln(err)
	}

	return time.Duration(total) * time.Second, nil
}

// VolToHuman converts float64 volume that is used by audio library to human
// readable form (0 - 100)
func (p *MPDPlayer) VolToHuman(volume float64) int {
	return int(volume*10) + 100
}

func (p *MPDPlayer) reconnect() (err error) {
	p.client, err = mpd.Dial("tcp", p.mpdPort)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

// Stop is the place to stop playing
func (p *MPDPlayer) Stop() (err error) {

	if p.client == nil {
		if err = p.reconnect(); err != nil {
			return tracerr.Wrap(err)
		}
	}

	if err = p.client.Stop(); err != nil {
		return tracerr.Wrap(err)
	}

	if err = p.client.Close(); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}
