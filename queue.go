// Copyright (C) 2020  Raziman

package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Queue struct {
	*tview.List
	SavedQueuePath string
	Items          []*AudioFile
}

// highlight the next item in the queue
func (q *Queue) next() {
	currIndex := q.GetCurrentItem()
	idx := currIndex + 1
	if currIndex == q.GetItemCount()-1 {
		idx = 0
	}
	q.SetCurrentItem(idx)
}

// highlight the previous item in the queue
func (q *Queue) prev() {
	currIndex := q.GetCurrentItem()
	q.SetCurrentItem(currIndex - 1)
}

// usually used with GetCurrentItem which can return -1 if
// no item highlighted
func (q *Queue) DeleteItem(index int) {
	if index != -1 {
		q.RemoveItem(index)

		var nItems []*AudioFile

		for i, v := range q.Items {
			if i == index {
				continue
			}

			nItems = append(nItems, v)
		}

		q.Items = nItems
		q.UpdateTitle()

	}
}

func (q *Queue) UpdateTitle() {

	var totalLength time.Duration

	for _, v := range q.Items {
		totalLength += v.Length
	}

	fmtTime := fmtDuration(totalLength)

	q.SetTitle(fmt.Sprintf("┤ Queue ├──┤%s├", fmtTime))

}

// gets the first item and remove it from the queue
// app.Draw() must be called after calling this function
func (q *Queue) Dequeue() (string, error) {

	if q.GetItemCount() == 0 {
		return "", errors.New("Empty list")
	}

	_, first := q.GetItemText(0)

	q.DeleteItem(0)
	q.UpdateTitle()

	return first, nil
}

// Add item to the list and returns the length of the queue
func (q *Queue) Enqueue(audioFile *AudioFile) int {

	q.Items = append(q.Items, audioFile)

	if !gomu.Player.IsRunning {

		gomu.Player.IsRunning = true

		go func() {
			// we dont need the primary text as it will be dequeued anyway
			q.AddItem("", audioFile.Path, 0, nil)
			gomu.Player.Run()
		}()

		return q.GetItemCount()

	}

	songLength, err := GetLength(audioFile.Path)

	if err != nil {
		appLog(err)
	}

	queueItemView := fmt.Sprintf("[ %s ] %s", fmtDuration(songLength), GetName(audioFile.Name))
	q.AddItem(queueItemView, audioFile.Path, 0, nil)
	q.UpdateTitle()

	return q.GetItemCount()
}

// GetItems is used to get the secondary text
// which is used to store the path of the audio file
// this is for the sake of convenience
func (q *Queue) GetItems() []string {

	items := []string{}

	for i := 0; i < q.GetItemCount(); i++ {

		_, second := q.GetItemText(i)

		items = append(items, second)
	}

	return items
}

// Save the current queue in a csv file
func (q *Queue) SaveQueue() error {

	songPaths := q.GetItems()
	songNames := make([]string, 0, len(songPaths))
	var content string

	for _, songPath := range songPaths {
		hashed := Sha1Hex(GetName(songPath))
		songNames = append(songNames, hashed)
	}

	for _, v := range songNames {
		content += v + "\n"
	}

	cachePath := expandTilde(q.SavedQueuePath)
	err := ioutil.WriteFile(cachePath, []byte(content), 0644)

	if err != nil {
		return err
	}

	return nil

}

// Clears current queue
func (q *Queue) ClearQueue() {

	q.Items = []*AudioFile{}
	q.Clear()
	q.UpdateTitle()

}

// Loads previously saved list
func (q *Queue) LoadQueue() error {

	songs, err := q.GetSavedQueue()

	if err != nil {
		return err
	}

	for _, v := range songs {

		audioFile := gomu.Playlist.FindAudioFile(v)

		if audioFile != nil {
			q.Enqueue(audioFile)
		}
	}

	return nil
}

// Get saved queue, if not exist, create it
func (q *Queue) GetSavedQueue() ([]string, error) {

	queuePath := expandTilde(q.SavedQueuePath)

	if _, err := os.Stat(queuePath); os.IsNotExist(err) {

		dir, _ := path.Split(queuePath)

		err := os.MkdirAll(dir, 0744)

		if err != nil {
			return nil, err
		}

		_, err = os.Create(queuePath)

		if err != nil {
			return nil, err
		}

		return []string{}, nil

	}

	f, err := os.Open(queuePath)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	records := []string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		records = append(records, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// Initiliaze new queue with default values
func NewQueue() *Queue {

	list := tview.NewList().
		ShowSecondaryText(false)

	queue := &Queue{
		List:           list,
		SavedQueuePath: "~/.local/share/gomu/queue.cache",
		Items:          []*AudioFile{},
	}

	queue.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Rune() {
		case 'j':
			queue.next()
		case 'k':
			queue.prev()
		case 'd':
			queue.DeleteItem(queue.GetCurrentItem())
		case 'D':
			queue.ClearQueue()
		}

		return nil
	})

	queue.UpdateTitle()
	queue.SetBorder(true).SetTitleAlign(tview.AlignLeft)
	queue.
		SetSelectedBackgroundColor(tcell.ColorDarkCyan).
		SetSelectedTextColor(tcell.ColorWhite).
		SetHighlightFullLine(true)

	return queue

}

func Sha1Hex(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}
