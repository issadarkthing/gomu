// Copyright (C) 2020  Raziman

package main

import (
	"errors"

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
func (q *Queue) Pop() (string, error) {

	if q.GetItemCount() == 0 {
		return "", errors.New("Empty list")
	}

	_, first := q.GetItemText(0)

	q.deleteItem(0)

	return first, nil
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
