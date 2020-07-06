// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Queue struct {
	*tview.List
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
func (q *Queue) deleteItem(index int) {
	if index != -1 {
		q.RemoveItem(index)
	}
}

// gets the first item and remove it from the queue
// app.Draw() must be called after calling this function
func (q *Queue) Dequeue() (string, error) {

	if q.GetItemCount() == 0 {
		return "", errors.New("Empty list")
	}

	_, first := q.GetItemText(0)

	q.deleteItem(0)

	return first, nil
}

// Add item to the list and returns the length of the queue
func (q *Queue) Enqueue(audioFile *AudioFile) int {

	if !gomu.Player.IsRunning {

		gomu.Player.IsRunning = true

		go func() {
			// we dont need the primary text as it will be popped anyway
			q.AddItem("", audioFile.Path, 0, nil)
			gomu.Player.Run()
		}()

		return q.GetItemCount()

	}

	songLength, err := GetLength(audioFile.Path)

	if err != nil {
		appLog(err)
	}

	queueItemView := fmt.Sprintf("[ %s ] %s", fmtDuration(songLength), audioFile.Name)
	q.AddItem(queueItemView, audioFile.Path, 0, nil)

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

func NewQueue() *Queue {

	list := tview.NewList().
		ShowSecondaryText(false)

	queue := &Queue{list}

	queue.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Rune() {
		case 'j':
			queue.next()
		case 'k':
			queue.prev()
		case 'd':
			queue.deleteItem(queue.GetCurrentItem())
		}

		return nil
	})

	queue.SetHighlightFullLine(true)
	queue.SetBorder(true).SetTitle(" Queue ")
	queue.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	queue.SetSelectedTextColor(tcell.ColorWhite)

	return queue

}
