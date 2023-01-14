// Copyright (C) 2020  Raziman

package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/player"
)

// Queue shows queued songs for playing
type Queue struct {
	*tview.List
	savedQueuePath string
	items          []*player.AudioFile
	isLoop         bool
}

// Highlight the next item in the queue
func (q *Queue) next() {
	currIndex := q.GetCurrentItem()
	idx := currIndex + 1
	if currIndex == q.GetItemCount()-1 {
		idx = 0
	}
	q.SetCurrentItem(idx)
}

// Highlight the previous item in the queue
func (q *Queue) prev() {
	currIndex := q.GetCurrentItem()
	q.SetCurrentItem(currIndex - 1)
}

// Usually used with GetCurrentItem which can return -1 if
// no item highlighted
func (q *Queue) deleteItem(index int) (*player.AudioFile, error) {

	if index > len(q.items)-1 {
		return nil, tracerr.New("Index out of range")
	}

	// deleted audio file
	var dAudio *player.AudioFile

	if index != -1 {
		q.RemoveItem(index)

		var nItems []*player.AudioFile

		for i, v := range q.items {

			if i == index {
				dAudio = v
				continue
			}

			nItems = append(nItems, v)
		}

		q.items = nItems
		// here we move to next item if not at the end
		if index < len(q.items) {
			q.next()
		}
		q.updateTitle()

	}

	return dAudio, nil
}

// Update queue title which shows number of items and total length
func (q *Queue) updateTitle() string {

	var totalLength time.Duration

	for _, v := range q.items {
		totalLength += v.Len()
	}

	fmtTime := fmtDurationH(totalLength)

	var count string

	if len(q.items) > 1 {
		count = "songs"
	} else {
		count = "song"
	}

	var loop string

	isEmoji := gomu.anko.GetBool("General.use_emoji")

	if q.isLoop {
		if isEmoji {
			loop = gomu.anko.GetString("Emoji.loop")
		} else {
			loop = "Loop"
		}
	} else {
		if isEmoji {
			loop = gomu.anko.GetString("Emoji.noloop")
		} else {
			loop = "No loop"
		}
	}

	title := fmt.Sprintf("─ Queue ───┤ %d %s | %s | %s ├",
		len(q.items), count, fmtTime, loop)

	q.SetTitle(title)

	return title
}

// Add item to the front of the queue
func (q *Queue) pushFront(audioFile *player.AudioFile) {

	q.items = append([]*player.AudioFile{audioFile}, q.items...)

	songLength := audioFile.Len()

	queueItemView := fmt.Sprintf(
		"[ %s ] %s", fmtDuration(songLength), getName(audioFile.Name()),
	)

	q.InsertItem(0, queueItemView, audioFile.Path(), 0, nil)
	q.updateTitle()
}

// gets the first item and remove it from the queue
// app.Draw() must be called after calling this function
func (q *Queue) dequeue() (*player.AudioFile, error) {

	if q.GetItemCount() == 0 {
		return nil, tracerr.New("Empty list")
	}

	first := q.items[0]
	q.deleteItem(0)
	q.updateTitle()

	return first, nil
}

// Add item to the list and returns the length of the queue
func (q *Queue) enqueue(audioFile *player.AudioFile) (int, error) {

	if !audioFile.IsAudioFile() {
		return q.GetItemCount(), nil
	}

	q.items = append(q.items, audioFile)
	songLength, err := getTagLength(audioFile.Path())

	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	queueItemView := fmt.Sprintf(
		"[ %s ] %s", fmtDuration(songLength), getName(audioFile.Name()),
	)
	q.AddItem(queueItemView, audioFile.Path(), 0, nil)
	q.updateTitle()

	return q.GetItemCount(), nil
}

// getItems is used to get the secondary text
// which is used to store the path of the audio file
// this is for the sake of convenience
func (q *Queue) getItems() []string {

	items := []string{}

	for i := 0; i < q.GetItemCount(); i++ {

		_, second := q.GetItemText(i)

		items = append(items, second)
	}

	return items
}

// Save the current queue
func (q *Queue) saveQueue() error {

	songPaths := q.getItems()
	var content strings.Builder

	if gomu.player.HasInit() && gomu.player.GetCurrentSong() != nil {
		currentSongPath := gomu.player.GetCurrentSong().Path()
		currentSongInQueue := false
		for _, songPath := range songPaths {
			if getName(songPath) == getName(currentSongPath) {
				currentSongInQueue = true
			}
		}
		if !currentSongInQueue && len(q.items) != 0 {
			hashed := sha1Hex(getName(currentSongPath))
			content.WriteString(hashed + "\n")
		}
	}

	for _, songPath := range songPaths {
		// hashed song name is easier to search through
		hashed := sha1Hex(getName(songPath))
		content.WriteString(hashed + "\n")
	}

	savedPath := expandTilde(q.savedQueuePath)
	err := ioutil.WriteFile(savedPath, []byte(content.String()), 0644)

	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil

}

// Clears current queue
func (q *Queue) clearQueue() {

	q.items = []*player.AudioFile{}
	q.Clear()
	q.updateTitle()

}

// Loads previously saved list
func (q *Queue) loadQueue() error {

	songs, err := q.getSavedQueue()

	if err != nil {
		return tracerr.Wrap(err)
	}

	for _, v := range songs {

		audioFile, err := gomu.playlist.findAudioFile(v)

		if err != nil {
			logError(err)
			continue
		}

		q.enqueue(audioFile)
	}

	return nil
}

// Get saved queue, if not exist, create it
func (q *Queue) getSavedQueue() ([]string, error) {

	queuePath := expandTilde(q.savedQueuePath)

	if _, err := os.Stat(queuePath); os.IsNotExist(err) {

		dir, _ := path.Split(queuePath)
		err := os.MkdirAll(dir, 0744)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		_, err = os.Create(queuePath)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		return []string{}, nil

	}

	f, err := os.Open(queuePath)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	records := []string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		records = append(records, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, tracerr.Wrap(err)
	}

	return records, nil
}

func (q *Queue) help() []string {

	return []string{
		"j      down",
		"k      up",
		"l      play selected song",
		"d      remove from queue",
		"D      clear queue",
		"z      toggle loop",
		"s      shuffle",
		"/      find in queue",
		"t      lyric delay increase 0.5 second",
		"r      lyric delay decrease 0.5 second",
	}

}

// Shuffles the queue
func (q *Queue) shuffle() {

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(q.items), func(i, j int) {
		q.items[i], q.items[j] = q.items[j], q.items[i]
	})

	q.Clear()

	for _, v := range q.items {
		audioLen, err := getTagLength(v.Path())
		if err != nil {
			logError(err)
		}

		queueText := fmt.Sprintf("[ %s ] %s", fmtDuration(audioLen), v.Name())
		q.AddItem(queueText, v.Path(), 0, nil)
	}

	// q.updateTitle()

}

// Initiliaze new queue with default values
func newQueue() *Queue {

	list := tview.NewList()
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logError(err)
	}
	cacheQueuePath := filepath.Join(cacheDir, "gomu", "queue.cache")
	queue := &Queue{
		List:           list,
		savedQueuePath: cacheQueuePath,
	}

	cmds := map[rune]string{
		'j': "move_down",
		'k': "move_up",
		'd': "delete_item",
		'D': "clear_queue",
		'l': "play_selected",
		'z': "toggle_loop",
		's': "shuffle_queue",
		'/': "queue_search",
		't': "lyric_delay_increase",
		'r': "lyric_delay_decrease",
	}

	for key, cmdName := range cmds {
		src := fmt.Sprintf(`Keybinds.def_q("%c", %s)`, key, cmdName)
		gomu.anko.Execute(src)
	}

	queue.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		if gomu.anko.KeybindExists("queue", e) {

			err := gomu.anko.ExecKeybind("queue", e)
			if err != nil {
				errorPopup(err)
			}

		}

		return nil
	})

	queue.updateTitle()

	queue.
		ShowSecondaryText(false).
		SetSelectedBackgroundColor(gomu.colors.queueHi).
		SetSelectedTextColor(gomu.colors.foreground).
		SetHighlightFullLine(true)

	queue.
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorderColor(gomu.colors.foreground).
		SetBackgroundColor(gomu.colors.background)

	return queue
}

// Convert string to sha1.
func sha1Hex(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// Modify the title of songs in queue
func (q *Queue) renameItem(oldAudio *player.AudioFile, newAudio *player.AudioFile) error {
	for i, v := range q.items {
		if v.Name() != oldAudio.Name() {
			continue
		}
		err := q.insertItem(i, newAudio)
		if err != nil {
			return tracerr.Wrap(err)
		}
		_, err = q.deleteItem(i + 1)
		if err != nil {
			return tracerr.Wrap(err)
		}

	}
	return nil
}

// playQueue play the first item in the queue
func (q *Queue) playQueue() error {

	audioFile, err := q.dequeue()
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = gomu.player.Run(audioFile)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func (q *Queue) insertItem(index int, audioFile *player.AudioFile) error {

	if index > len(q.items)-1 {
		return tracerr.New("Index out of range")
	}

	if index != -1 {
		songLength, err := getTagLength(audioFile.Path())
		if err != nil {
			return tracerr.Wrap(err)
		}
		queueItemView := fmt.Sprintf(
			"[ %s ] %s", fmtDuration(songLength), getName(audioFile.Name()),
		)

		q.InsertItem(index, queueItemView, audioFile.Path(), 0, nil)

		var nItems []*player.AudioFile

		for i, v := range q.items {

			if i == index {
				nItems = append(nItems, audioFile)
			}

			nItems = append(nItems, v)
		}

		q.items = nItems
		q.updateTitle()

	}

	return nil
}

// update the path information in queue
func (q *Queue) updateQueuePath() {

	var songs []string
	if len(q.items) < 1 {
		return
	}
	for _, v := range q.items {
		song := sha1Hex(getName(v.Name()))
		songs = append(songs, song)
	}

	q.clearQueue()
	for _, v := range songs {

		audioFile, err := gomu.playlist.findAudioFile(v)

		if err != nil {
			continue
		}
		q.enqueue(audioFile)
	}

	q.updateTitle()
}

// update current playing song name to reflect the changes during rename and paste
func (q *Queue) updateCurrentSongName(oldAudio *player.AudioFile, newAudio *player.AudioFile) error {

	if !gomu.player.IsRunning() && !gomu.player.IsPaused() {
		return nil
	}

	currentSong := gomu.player.GetCurrentSong()
	position := gomu.playingBar.getProgress()
	paused := gomu.player.IsPaused()

	if oldAudio.Name() != currentSong.Name() {
		return nil
	}

	// we insert it in the first of queue, then play it
	gomu.queue.pushFront(newAudio)
	tmpLoop := q.isLoop
	q.isLoop = false
	gomu.player.Skip()
	gomu.player.Seek(position)
	if paused {
		gomu.player.TogglePause()
	}
	q.isLoop = tmpLoop
	q.updateTitle()

	return nil
}

// update current playing song path to reflect the changes during rename and paste
func (q *Queue) updateCurrentSongPath(oldAudio *player.AudioFile, newAudio *player.AudioFile) error {

	if !gomu.player.IsRunning() && !gomu.player.IsPaused() {
		return nil
	}

	currentSong := gomu.player.GetCurrentSong()
	position := gomu.playingBar.getProgress()
	paused := gomu.player.IsPaused()

	// Here we check the situation when currentsong is under oldAudio folder
	if !strings.Contains(currentSong.Path(), oldAudio.Path()) {
		return nil
	}

	// Here is the handling of folder rename and paste
	currentSongAudioFile, err := gomu.playlist.findAudioFile(sha1Hex(getName(currentSong.Name())))
	if err != nil {
		return tracerr.Wrap(err)
	}
	gomu.queue.pushFront(currentSongAudioFile)
	tmpLoop := q.isLoop
	q.isLoop = false
	gomu.player.Skip()
	gomu.player.Seek(position)
	if paused {
		gomu.player.TogglePause()
	}
	q.isLoop = tmpLoop

	q.updateTitle()
	return nil

}

// update current playing song simply delete it
func (q *Queue) updateCurrentSongDelete(oldAudio *player.AudioFile) {
	if !gomu.player.IsRunning() && !gomu.player.IsPaused() {
		return
	}

	currentSong := gomu.player.GetCurrentSong()
	paused := gomu.player.IsPaused()

	var delete bool
	if oldAudio.IsAudioFile() {
		if oldAudio.Name() == currentSong.Name() {
			delete = true
		}
	} else {
		if strings.Contains(currentSong.Path(), oldAudio.Path()) {
			delete = true
		}
	}

	if !delete {
		return
	}

	tmpLoop := q.isLoop
	q.isLoop = false
	gomu.player.Skip()
	if paused {
		gomu.player.TogglePause()
	}
	q.isLoop = tmpLoop
	q.updateTitle()

}
