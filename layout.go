package main

import "github.com/rivo/tview"

func Layout(
	app *tview.Application,
	child1 *tview.TreeView,
	child2 *tview.List,
	child3 *tview.Box,
) *tview.Flex {

	flex := tview.NewFlex().
		AddItem(child1, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(child2, 0, 7, false).
			AddItem(child3, 0, 1, false), 0, 3, false)

	return flex

}
