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
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

type Queue struct {
	*tview.List
	savedQueuePath string
	items          []*AudioFile
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
func (q *Queue) deleteItem(index int) (*AudioFile, error) {

	if index > len(q.items)-1 {
		return nil, tracerr.New("Index out of range")
	}

	// deleted audio file
	var dAudio *AudioFile

	if index != -1 {
		q.RemoveItem(index)

		var nItems []*AudioFile

		for i, v := range q.items {

			if i == index {
				dAudio = v
				continue
			}

			nItems = append(nItems, v)
		}

		q.items = nItems
		q.updateTitle()

	}

	return dAudio, nil
}

// Update queue title which shows number of items and total length
func (q *Queue) updateTitle() string {

	var totalLength time.Duration

	for _, v := range q.items {
		totalLength += v.length
	}

	fmtTime := fmtDurationH(totalLength)

	var count string

	if len(q.items) > 1 {
		count = "songs"
	} else {
		count = "song"
	}

  var loop string

  if q.isLoop {
    loop = "ﯩ"
  } else {
    loop = ""
  }

	title := fmt.Sprintf("─ Queue ───┤ %d %s | %s | %s ├",
		len(q.items), count, fmtTime, loop)

	q.SetTitle(title)

	return title
}

// Add item to the front of the queue
func (q *Queue) pushFront(audioFile *AudioFile) {

	q.items = append([]*AudioFile{audioFile}, q.items...)

	songLength := audioFile.length

	queueItemView := fmt.Sprintf(
		"[ %s ] %s", fmtDuration(songLength), getName(audioFile.name),
	)

	q.InsertItem(0, queueItemView, audioFile.path, 0, nil)
	q.updateTitle()
}

// gets the first item and remove it from the queue
// app.Draw() must be called after calling this function
func (q *Queue) dequeue() (*AudioFile, error) {

	if q.GetItemCount() == 0 {
		return nil, tracerr.New("Empty list")
	}

	first := q.items[0]
	q.deleteItem(0)
	q.updateTitle()

	return first, nil
}

// Add item to the list and returns the length of the queue
func (q *Queue) enqueue(audioFile *AudioFile) (int, error) {

	if !gomu.player.isRunning && os.Getenv("TEST") == "false" {

		gomu.player.isRunning = true

		go func() {

			if err := gomu.player.run(audioFile); err != nil {
				logError(err)
			}

		}()

		return q.GetItemCount(), nil

	}

  if ! audioFile.isAudioFile {
    return q.GetItemCount(), nil
  } 

	q.items = append(q.items, audioFile)
	songLength, err := getLength(audioFile.path)

	if err != nil {
		return 0, tracerr.Wrap(err)
	}

	queueItemView := fmt.Sprintf(
		"[ %s ] %s", fmtDuration(songLength), getName(audioFile.name),
	)
	q.AddItem(queueItemView, audioFile.path, 0, nil)
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


  currentSongPath := gomu.player.currentSong.path
  currentSongInQueue := false
 	for _, songPath := range songPaths {
    if songPath == currentSongPath {
      currentSongInQueue = true
    }
	}
  if ! currentSongInQueue {
    hashed := sha1Hex(getName(currentSongPath))
    content.WriteString(hashed + "\n")
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

	q.items = []*AudioFile{}
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

	defer f.Close()

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

// Fuzzy find queue
func (q *Queue) fuzzyFind() error {

	var result string
	var err error

	audioFiles := q.items
	input := make([]string, 0, len(audioFiles))

	for _, v := range q.items {
		input = append(input, v.name)
	}

	ok := gomu.app.Suspend(func() {
		result, err = fzfFind(input)
	})

	if err != nil {
		return tracerr.Wrap(err)
	}

	if !ok {
		return tracerr.New("Fzf not executed")
	}

	var index int
	for i, v := range q.items {
		if v.name == result {
			index = i
		}
	}

	if result == "" {
		return nil
	}

	q.SetCurrentItem(index)

	return nil
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
		"f      find in queue",
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
		audioLen, err := getLength(v.path)
		if err != nil {
			logError(err)
		}

		queueText := fmt.Sprintf("[ %s ] %s", fmtDuration(audioLen), v.name)
		q.AddItem(queueText, v.path, 0, nil)
	}

	q.updateTitle()

}

// Initiliaze new queue with default values
func newQueue() *Queue {

	list := tview.NewList().
		ShowSecondaryText(false)

	queue := &Queue{
		List:           list,
		savedQueuePath: "~/.local/share/gomu/queue.cache",
	}

	queue.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		cmds := map[rune]string{
			'j': "move_down",
			'k': "move_up",
			'd': "delete_item",
			'D': "clear_queue",
			'l': "play_selected",
			'z': "toggle_loop",
			's': "shuffle_queue",
			'/': "queue_search",
		}

		for key, cmd := range cmds {
			if e.Rune() != key {
				continue
			}
			fn, err := gomu.command.getFn(cmd)
			if err != nil {
				logError(err)
				return e
			}
			fn()
		}

		return nil
	})

	queue.updateTitle()
	queue.SetBorder(true).SetTitleAlign(tview.AlignLeft)
	queue.
		SetSelectedBackgroundColor(tcell.ColorDarkCyan).
		SetSelectedTextColor(tcell.ColorWhite).
		SetHighlightFullLine(true).
		SetBorderPadding(0, 0, 1, 1)

	return queue

}

// Convert string to sha1.
func sha1Hex(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}
