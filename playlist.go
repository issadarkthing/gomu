// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Parent      *tview.TreeNode
}

type Playlist struct {
	*tview.TreeView
}

func InitPlaylist() *Playlist {

	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		log(err.Error())
	}

	root := tview.NewTreeNode(path.Base(rootDir))

	tree := tview.NewTreeView().SetRoot(root)
	playlist := &Playlist{tree}

	playlist.SetTitle(" Playlist ").SetBorder(true)

	populate(root, rootDir)

	var firstChild *tview.TreeNode
	var prevNode *tview.TreeNode

	if len(root.GetChildren()) == 0 {
		firstChild = root
	} else {
		firstChild = root.GetChildren()[0]
	}

	firstChild.SetColor(textColor)
	playlist.SetCurrentNode(firstChild)
	// keep track of prev node so we can remove the color of highlight
	prevNode = firstChild.SetColor(accentColor)

	playlist.SetChangedFunc(func(node *tview.TreeNode) {
		prevNode.SetColor(textColor)
		root.SetColor(textColor)
		node.SetColor(accentColor)
		prevNode = node
	})

	playlist.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := playlist.GetCurrentNode()

		if currNode == playlist.GetRoot() {
			return e
		}

		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {
		case 'l':

			playlist.addToQueue(audioFile)
			currNode.SetExpanded(true)

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				currNode.SetColor(textColor)
				parent.SetExpanded(false)
				parent.SetColor(accentColor)
				// prevPanel = parent
				playlist.SetCurrentNode(parent)
			}

			currNode.Collapse()

		case 'L':

			if !viper.GetBool("confirm_bulk_add") {
				playlist.addAllToQueue(playlist.GetCurrentNode())
				return e
			}

			confirmationPopup(
				"Are you sure to add this whole directory into queue?",
				func(_ int, label string) {

					if label == "yes" {
						playlist.addAllToQueue(playlist.GetCurrentNode())
					}

					pages.RemovePage("confirmation-popup")
					app.SetFocus(playlist)

				})

		}

		return e
	})

	return playlist

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

// add to queue and update queue panel
func (playlist *Playlist) addToQueue(audioFile *AudioFile) {

	if audioFile.IsAudioFile {

		if !player.IsRunning {

			player.IsRunning = true

			go func() {
				queue.AddItem("", audioFile.Path, 0, nil)
				player.Run()
			}()

		} else {

			songLength, err := GetLength(audioFile.Path)

			if err != nil {
				log(err.Error())
			}

			queueItemView := fmt.Sprintf("[ %s ] %s", fmtDuration(songLength), audioFile.Name)
			queue.AddItem(queueItemView, audioFile.Path, 0, nil)
		}
	}
}

// bulk add a playlist to queue
func (playlist *Playlist) addAllToQueue(root *tview.TreeNode) {

	var childrens []*tview.TreeNode

	childrens = root.GetChildren()

	// gets the parent if highlighted item is a file
	if len(childrens) == 0 {
		childrens = root.GetReference().(*AudioFile).Parent.GetChildren()
	}

	for _, v := range childrens {
		currNode := v.GetReference().(*AudioFile)

		playlist.addToQueue(currNode)
	}

}
