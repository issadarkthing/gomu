// Copyright (C) 2020  Raziman

package player

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rivo/tview"
	"github.com/tramhao/id3v2"
	"github.com/ztrue/tracerr"
)

var _ Audio = (*AudioFile)(nil)

// AudioFile represents directories and mp3 files
// isAudioFile equals to false if it is a directory
type AudioFile struct {
	name        string
	path        string
	isAudioFile bool
	length      time.Duration
	node        *tview.TreeNode
	parent      *tview.TreeNode
}

// NewAudioFile return a new instance of Audiofile
func NewAudioFile() *AudioFile {
	return &AudioFile{}
}

// Name return the name of AudioFile
func (a *AudioFile) Name() string {
	return a.name
}

// SetName set the name of AudioFile
func (a *AudioFile) SetName(name string) {
	if name == "" {
		return
	}
	a.name = name
}

// Path return the path of AudioFile
func (a *AudioFile) Path() string {
	return a.path
}

// SetPath return the path of AudioFile
func (a *AudioFile) SetPath(path string) {
	a.path = path
}

// IsAudioFile check if the file is song or directory
func (a *AudioFile) IsAudioFile() bool {
	return a.isAudioFile
}

// SetIsAudioFile check if the file is song or directory
func (a *AudioFile) SetIsAudioFile(isAudioFile bool) {
	a.isAudioFile = isAudioFile
}

// Len return the length of AudioFile
func (a *AudioFile) Len() time.Duration {
	return a.length
}

// SetLen set the length of AudioFile
func (a *AudioFile) SetLen(length time.Duration) {
	a.length = length
}

// Parent return the parent directory of AudioFile
func (a *AudioFile) Parent() *AudioFile {
	if a.parent == nil {
		return nil
	}
	return a.parent.GetReference().(*AudioFile)
}

// SetParentNode return the parent directory of AudioFile
func (a *AudioFile) SetParentNode(parentNode *tview.TreeNode) {
	if parentNode == nil {
		return
	}
	a.parent = parentNode
}

// ParentNode return the parent node of AudioFile
func (a *AudioFile) ParentNode() *tview.TreeNode {
	if a.parent == nil {
		return nil
	}
	return a.parent
}

// Node return the current node of AudioFile
func (a *AudioFile) Node() *tview.TreeNode {
	if a.node == nil {
		return nil
	}
	return a.node
}

// SetNode return the current node of AudioFile
func (a *AudioFile) SetNode(node *tview.TreeNode) {
	a.node = node
}

// String return the string of AudioFile
func (a *AudioFile) String() string {
	if a == nil {
		return "nil"
	}
	return fmt.Sprintf("%#v", a)
}

// LoadTagMap will load from tag and return a map of langExt to lyrics
func (a *AudioFile) LoadTagMap() (tag *id3v2.Tag, popupLyricMap map[string]string, options []string, err error) {

	popupLyricMap = make(map[string]string)

	if a.isAudioFile {
		tag, err = id3v2.Open(a.path, id3v2.Options{Parse: true})
		if err != nil {
			return nil, nil, nil, tracerr.Wrap(err)
		}
		defer tag.Close()
	} else {
		return nil, nil, nil, fmt.Errorf("not an audio file")
	}
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range usltFrames {
		uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			return nil, nil, nil, errors.New("USLT error")
		}
		res := uslf.Lyrics
		popupLyricMap[uslf.ContentDescriptor] = res
	}
	for option := range popupLyricMap {
		options = append(options, option)
	}
	sort.Strings(options)

	return tag, popupLyricMap, options, err
}
