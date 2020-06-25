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

var (
	textColor       = tcell.ColorWhite
	accentColor     = tcell.ColorDarkCyan
)

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Parent      *tview.TreeNode
}

func Playlist(player *Player) *tview.TreeView {


	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		log(err.Error())
	}

	root := tview.NewTreeNode(path.Base(rootDir))

	tree := tview.NewTreeView().SetRoot(root)
	tree.SetTitle(" Playlist ").SetBorder(true)

	var prevNode *tview.TreeNode


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
	prevNode = firstChild.SetColor(accentColor)

	tree.SetChangedFunc(func(node *tview.TreeNode) {

		prevNode.SetColor(textColor)
		root.SetColor(textColor)
		node.SetColor(accentColor)
		prevNode = node
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

func addToQueue(audioFile *AudioFile, player *Player, list *tview.List) {


	if audioFile.IsAudioFile {

		player.Push(audioFile.Path)

		if !player.IsRunning {

			player.IsRunning = true

			go func () {
				player.Run()
				list.AddItem("", "", 0, nil)
			} ()

		} else {

			songLength, err := player.GetLength(len(player.queue) - 1)

			if err != nil {
				log(err.Error())
			}
			list.AddItem(
				fmt.Sprintf("[ %s ] %s", fmtDuration(songLength), audioFile.Name), 
				"", 0, nil)
			}
	}
}

func addAllToQueue(root *tview.TreeNode, player *Player, list *tview.List) {

	var childrens []*tview.TreeNode

	childrens = root.GetChildren()

	// gets the parent if highlighted item is a file
	if len(childrens) == 0 {
		childrens = root.GetReference().(*AudioFile).Parent.GetChildren()
	} 

	for _, v := range childrens {

		currNode := v.GetReference().(*AudioFile)

		addToQueue(currNode, player, list)

	}

}
