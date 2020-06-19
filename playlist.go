package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)


type AudioFile struct {
	Name string
	Path string
	IsAudioFile bool
	Parent *tview.TreeNode
}

func playlist(list *tview.List) *tview.TreeView {

	musicDir := "./music"

	rootDir, err := filepath.Abs(musicDir)

	if err != nil {
		panic(err)
	}

	root := tview.NewTreeNode(musicDir)

	tree := tview.NewTreeView().SetRoot(root)
	tree.SetTitle("Playlist").SetBorder(true)
	tree.SetGraphicsColor(tcell.ColorWhite)

	textColor := tcell.ColorAntiqueWhite
	backGroundColor := tcell.ColorDarkCyan
	var prevNode *tview.TreeNode

	go func() {

		populate(root, rootDir)

		firstChild := root.GetChildren()[0]

		firstChild.SetColor(textColor)
		tree.SetCurrentNode(firstChild)
		// keep track of prev node so we can remove the color of highlight
		prevNode = firstChild.SetColor(backGroundColor)

		tree.SetChangedFunc(func (node *tview.TreeNode) {

			prevNode.SetColor(textColor)
			root.SetColor(textColor)
			node.SetColor(backGroundColor)
			prevNode = node
		})

	} ()


	tree.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := tree.GetCurrentNode()

		if currNode == root {
			return e
		}
		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {
		case 'l':

			if audioFile.IsAudioFile {

				list.AddItem(audioFile.Name, audioFile.Path, 0, nil)
				
			}
			currNode.SetExpanded(true)
		case 'h':

			// if closing node with no children
			// close the node's parent 
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				currNode.SetColor(textColor)
				parent.SetExpanded(false)
				parent.SetColor(backGroundColor)
				prevNode = parent
				tree.SetCurrentNode(parent)
			} else {
				currNode.Collapse()
			}

		}

		return e
	})

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	return tree

}

func populate(root *tview.TreeNode, rootPath string) {

	files, err := ioutil.ReadDir(rootPath)

	if err != nil {
		panic(err)
	}

	for _, file := range files {

		path := filepath.Join(rootPath, file.Name())
		child := tview.NewTreeNode(file.Name())
		root.AddChild(child)

		audioFile := &AudioFile{
			Name: file.Name(),
			Path: path,
			IsAudioFile: true,
			Parent: root,	
		}

		child.SetReference(audioFile)

		if file.IsDir() {
			audioFile.IsAudioFile = false
			populate(child, path)
		}

	}

}

