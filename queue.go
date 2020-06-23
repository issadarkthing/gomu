// Copyright (C) 2020  Raziman

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func Queue(player *Player) *tview.List {

	list := tview.NewList().
		ShowSecondaryText(false)

	next := func() {

		currIndex := list.GetCurrentItem()
		idx := currIndex + 1
		if currIndex == list.GetItemCount()-1 {
			idx = 0
		}
		list.SetCurrentItem(idx)
	}

	prev := func() {
		currIndex := list.GetCurrentItem()
		list.SetCurrentItem(currIndex - 1)
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		switch e.Rune() {
		case 'j':
			next()
		case 'k':
			prev()
		case 'd':
			index := list.GetCurrentItem()

			if index != -1 {
				player.Remove(index)
				list.RemoveItem(index)
			}

		}

		return nil
	})



	list.SetHighlightFullLine(true)
	list.SetBorder(true).SetTitle(" Queue ")
	list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	list.SetSelectedTextColor(tcell.ColorWhite)

	return list

}

