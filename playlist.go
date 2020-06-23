// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	musicDir        = "./music"
	textColor       = tcell.ColorWhite
	backGroundColor = tcell.ColorDarkCyan
)

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Parent      *tview.TreeNode
}

func Playlist(list *tview.List, playBar *Progress, player *Player) *tview.TreeView {

	rootDir, err := filepath.Abs(musicDir)

	if err != nil {
		log(err.Error())
	}

	root := tview.NewTreeNode(musicDir)

	tree := tview.NewTreeView().SetRoot(root)
	tree.SetTitle(" Playlist ").SetBorder(true)
	tree.SetGraphicsColor(tcell.ColorWhite)

	var prevNode *tview.TreeNode

	go func() {

		populate(root, rootDir)

		var firstChild *tview.TreeNode

		if len(root.GetChildren()) == 0 {
			firstChild = root
		} else {
			firstChild = root.GetChildren()[0]
		}

		firstChild.SetColor(textColor)
		tree.SetCurrentNode(firstChild)
		// keep track of prev node so we can remove the color of highlight
		prevNode = firstChild.SetColor(backGroundColor)

		tree.SetChangedFunc(func(node *tview.TreeNode) {

			prevNode.SetColor(textColor)
			root.SetColor(textColor)
			node.SetColor(backGroundColor)
			prevNode = node
		})

	}()

	tree.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := tree.GetCurrentNode()

		if currNode == root {
			return e
		}
		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {
		case 'l':

			if audioFile.IsAudioFile {

				player.Push(audioFile.Path)

				if !player.IsRunning {

					go func () {
						player.Run()
						list.AddItem(
							fmt.Sprintf("%s | %s", player.length.String(), audioFile.Name),
							"", 0, nil)
					} ()

				} else {

					songLength, err := player.GetLength(len(player.queue) - 1)

					if err != nil {
						log(err.Error())
					}
					list.AddItem(
						fmt.Sprintf("[ %s ] %s", songLength.Round(time.Second).String(), audioFile.Name), 
						"", 0, nil)
				}
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
			}

			currNode.Collapse()

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
		log(err.Error())
	}

	for _, file := range files {

		path := filepath.Join(rootPath, file.Name())
		f, err := os.Open(path)

		if err != nil {
			log(err.Error())
		}

		defer f.Close()

		if !file.IsDir() {

			filetype, err := GetFileContentType(f)

			if err != nil {
				log(err.Error())
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

		}

		child := tview.NewTreeNode(file.Name())
		root.AddChild(child)

		audioFile := &AudioFile{
			Name:        file.Name(),
			Path:        path,
			IsAudioFile: true,
			Parent:      root,
		}

		child.SetReference(audioFile)

		if file.IsDir() {
			audioFile.IsAudioFile = false
			populate(child, path)
		}

	}

}
