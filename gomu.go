package main

import (
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func main() {

	app := tview.NewApplication()

	start(app)

}


func start(app *tview.Application) {
	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.BorderColor = tcell.ColorAntiqueWhite

	child1, child2, child3 := playlist(), queue(), nowPlayingBar()

	flex := layout(app, child1, child2, child3)

	pages := tview.NewPages().AddPage("main", flex, true, true)

	childrens := []Children{child1, child2, child3}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			cycleChildren(app, childrens)

		}

		switch event.Rune() {
		case 'q':
			confirmationPopup(app, pages, "Are you sure to exit?")
		}

		return event
	})


	// fix transparent background issue
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	// main loop
	if err := app.SetRoot(pages, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}

func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, false).
				AddItem(nil, 0, 1, false), width, 1, false).
			AddItem(nil, 0, 1, false)
}


func cycleChildren(app *tview.Application, childrens []Children) {

	focusedColor := tcell.ColorDarkCyan
	unfocusedColor := tcell.ColorAntiqueWhite
	anyChildHasFocus := false

	for i, child := range childrens {

		if child.HasFocus() {

			anyChildHasFocus = true

			var nextChild Children

			// if its the last element set the child back to one
			if i == len(childrens) - 1 {
				nextChild = childrens[0]
			} else {
				nextChild = childrens[i + 1]
			}


			child.SetBorderColor(unfocusedColor)	
			child.SetTitleColor(unfocusedColor)

			app.SetFocus(nextChild.(tview.Primitive))
			nextChild.SetBorderColor(focusedColor)
			nextChild.SetTitleColor(focusedColor)

			break
		}
	}

	if anyChildHasFocus == false {
		
		app.SetFocus(childrens[0].(tview.Primitive))
		childrens[0].SetBorderColor(focusedColor)
		childrens[0].SetTitleColor(focusedColor)
	}

}

// created so we can keep track of childrens in slices
type Children interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
}


func layout(
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

func nowPlayingBar() *tview.Box {
	return tview.NewBox().SetBorder(true).
		SetTitle("Currently Playing")
}

func queue() *tview.List {

	list := tview.NewList().
		AddItem("Lorem", "ipsum", '1', nil).
		AddItem("Lorem", "ipsum", '2', nil).
		AddItem("Lorem", "ipsum", '3', nil).
		AddItem("Lorem", "ipsum", '4', nil).
		AddItem("Lorem", "ipsum", '5', nil).
		ShowSecondaryText(false)

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

			switch e.Rune() {
			case 'j':
				next(list)
			case 'k':
				prev(list)
			}

			return nil
		})

	list.SetHighlightFullLine(true)
	list.SetBorder(true).SetTitle("Queue")
	list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	list.SetSelectedTextColor(tcell.ColorAntiqueWhite)

	return list

}

func confirmationPopup(app *tview.Application, pages *tview.Pages, text string) bool {

	result := false

	modal := tview.NewModal().
				SetText(text).
				SetBackgroundColor(tcell.ColorDefault).
				AddButtons([]string{"yes", "no"}).
				SetButtonBackgroundColor(tcell.ColorBlack).
				SetDoneFunc(func(_ int, buttonLabel string) {

					if buttonLabel == "yes" {
						result = true
						app.Stop()
					} else {
						pages.RemovePage("exit-popup")
					}

				});


	pages.AddPage("exit-popup", center(modal, 40, 10), true, true)
	app.SetFocus(modal)

	return result

}

func next(l *tview.List) {
	currIndex := l.GetCurrentItem()
	idx := currIndex + 1
	if currIndex == l.GetItemCount() - 1 {
		idx = 0
	}
	l.SetCurrentItem(idx)
}

func prev(l *tview.List) {
	currIndex := l.GetCurrentItem()
	l.SetCurrentItem(currIndex - 1)
}

func playlist() *tview.TreeView {

	rootDir := "~/Music"
	root := tview.NewTreeNode(rootDir)
	tree := tview.NewTreeView().SetRoot(root)
	tree.SetTitle("Playlist").SetBorder(true)
	tree.SetGraphicsColor(tcell.ColorWhite)

	textColor := tcell.ColorAntiqueWhite
	backGroundColor := tcell.ColorDarkCyan

	sample := [][]string{
		{"pop", "1", "2", "3"},
		{"rock", "1", "2", "3"},
		{"rap", "1", "2", "3"},
		{"country", "1", "2", "3"},
		{"edm", "1", "2", "3"},
	}

	var childrens []*tview.TreeNode

	// populate tree with sample
	for _, s := range sample {

		node := tview.NewTreeNode(s[0])

		node.SetExpanded(false)

		for _, sy := range s[1:] {

			child := tview.NewTreeNode(sy)
			child.SetIndent(5)
			child.SetReference(node)
			node.AddChild(child)

		}
		childrens = append(childrens, node)
		root.AddChild(node)
	}

	childrens[0].SetColor(backGroundColor)

	tree.SetCurrentNode(childrens[0])
	// keep track of prev node so we can remove the color of highlight
	prevNode := childrens[0].SetColor(backGroundColor)

	tree.SetChangedFunc(func (node *tview.TreeNode) {

		prevNode.SetColor(textColor)
		root.SetColor(textColor)
		node.SetColor(backGroundColor)
		prevNode = node
	})

	tree.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := tree.GetCurrentNode()

		switch e.Rune() {
		case 'l':
			currNode.SetExpanded(true)
		case 'h':

			// if closing node with no children
			// close the node's parent 
			if parent := currNode.GetReference(); parent != nil {
				// remove the color of the node
				currNode.SetColor(textColor)
				parent.(*tview.TreeNode).SetExpanded(false)
				parent.(*tview.TreeNode).SetColor(backGroundColor)
				prevNode = parent.(*tview.TreeNode)
				tree.SetCurrentNode(parent.(*tview.TreeNode))
			} else {
				currNode.SetExpanded(false)
			}
		}

		return e
	})

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	return tree

}

func log(text string) {

	f, err := os.OpenFile("message.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	if _, err := f.Write([]byte(text)); err != nil {
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

}
