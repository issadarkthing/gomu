// Package player is the place actually play the music
package player

import (
	"time"
)

type Audio interface {
	Name() string
	Path() string
}

type Player interface {
	// New returns new Player instance.
	//New(volume int) *BeepPlayer
	// SetSongFinish accepts callback which will be executed when the song finishes.
	SetSongFinish(f func(Audio))
	// SetSongStart accepts callback which will be executed when the song starts.
	SetSongStart(f func(Audio))
	// SetSongSkip accepts callback which will be executed when the song is skipped.
	SetSongSkip(f func(Audio))
	// executes songFinish callback.
	execSongFinish(a Audio)
	// executes songStart callback.
	execSongStart(a Audio)
	// executes songFinish callback.
	execSongSkip(a Audio)
	// Run plays the passed Audio.
	Run(currSong Audio) error
	// Pause pauses Player.
	Pause()
	// Play unpauses Player.
	Play()
	// SetVolume set volume up and volume down using -0.5 or +0.5.
	SetVolume(v float64) float64
	// TogglePause toggles the pause state.
	TogglePause()
	// Skip current song.
	Skip()
	// GetPosition returns the current position of audio file.
	GetPosition() time.Duration
	// Seek is the function to move forward and rewind
	Seek(pos int) error
	// IsPaused is used to distinguish the player between pause and stop
	IsPaused() bool
	// GetVolume returns current volume.
	GetVolume() float64
	// GetCurrentSong returns current song.
	GetCurrentSong() Audio
	// HasInit checks if the speaker has been initialized or not. Speaker
	// initialization will only happen once.
	HasInit() bool
	// IsRunning returns true if Player is running an audio.
	IsRunning() bool
	// GetLength return the length of the song in the queue
	GetLength(audioPath string) (time.Duration, error)
	// VolToHuman converts float64 volume that is used by audio library to human
	// readable form (0 - 100)
	VolToHuman(volume float64) int
	// AbsVolume converts human readable form volume (0 - 100) to float64 volume
	// that is used by the audio library
	//	AbsVolume(volume int) float64
	Stop() error
}

func NewPlayer(volume int, player string, mpdPort string) (Player, error) {
	switch player {
	case "mpd":
		return NewMPDPlayer(volume, mpdPort)
	default:
		return NewBeepPlayer(volume)
	}

}
