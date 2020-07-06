// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Length      time.Duration
	Parent      *tview.TreeNode
}

type Playlist struct {
	*tview.TreeView
	prevNode *tview.TreeNode
}

func NewPlaylist() *Playlist {

	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		appLog(err)
	}

	root := tview.NewTreeNode(path.Base(rootDir)).
		SetColor(gomu.AccentColor)

	tree := tview.NewTreeView().SetRoot(root)

	playlist := &Playlist{tree, nil}

	rootAudioFile := &AudioFile{
		Name:        root.GetText(),
		Path:        rootDir,
		IsAudioFile: false,
		Parent:      nil,
	}

	root.SetReference(rootAudioFile)

	playlist.SetTitle(" Playlist ").SetBorder(true)

	populate(root, rootDir)

	var firstChild *tview.TreeNode

	if len(root.GetChildren()) == 0 {
		firstChild = root
	} else {
		firstChild = root.GetChildren()[0]
	}

	playlist.SetHighlight(firstChild)

	playlist.SetChangedFunc(func(node *tview.TreeNode) {
		playlist.SetHighlight(node)
	})

	playlist.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := playlist.GetCurrentNode()

		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {

		case 'a':

			name, _ := gomu.Pages.GetFrontPage()

			if name != "mkdir-popup" {
				CreatePlaylistPopup()
			}

		case 'D':

			var selectedDir *AudioFile

			// gets the parent dir if current focused node is not a dir
			if audioFile.IsAudioFile {
				selectedDir = audioFile.Parent.GetReference().(*AudioFile)
			} else {
				selectedDir = audioFile
			}

			confirmationPopup(
				"Are you sure to delete this directory?", func(_ int, buttonName string) {

					if buttonName == "no" {
						gomu.Pages.RemovePage("confirmation-popup")
						gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
						return
					}

					err := os.RemoveAll(selectedDir.Path)

					if err != nil {
						timedPopup(
							" Error ",
							"Unable to delete dir "+selectedDir.Name, getPopupTimeout())
					} else {
						timedPopup(
							" Success ",
							selectedDir.Name+"\nhas been deleted successfully", getPopupTimeout())

						playlist.Refresh()
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

				})

		case 'd':

			// prevent from deleting a directory
			if !audioFile.IsAudioFile {
				return e
			}

			confirmationPopup(
				"Are you sure to delete this audio file?", func(_ int, buttonName string) {

					if buttonName == "no" {
						gomu.Pages.RemovePage("confirmation-popup")
						gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
						return
					}

					err := os.Remove(audioFile.Path)

					if err != nil {
						timedPopup(
							" Error ", "Unable to delete "+audioFile.Name, getPopupTimeout())
					} else {
						timedPopup(
							" Success ",
							audioFile.Name+"\nhas been deleted successfully", getPopupTimeout())

						playlist.Refresh()
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
				})

		case 'Y':

			if gomu.Pages.HasPage("download-popup") {
				gomu.Pages.RemovePage("download-popup")
				return e
			}

			// this ensures it downloads to
			// the correct dir
			if audioFile.IsAudioFile {
				downloadMusicPopup(audioFile.Parent)
			} else {
				downloadMusicPopup(currNode)
			}

		case 'l':

			gomu.Queue.Enqueue(audioFile)
			currNode.SetExpanded(true)

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				playlist.SetHighlight(parent)

				parent.SetExpanded(false)
			}

			currNode.Collapse()

		case 'L':

			if !viper.GetBool("confirm_bulk_add") {
				playlist.AddAllToQueue(playlist.GetCurrentNode())
				return e
			}

			confirmationPopup(
				"Are you sure to add this whole directory into queue?",
				func(_ int, label string) {

					if label == "yes" {
						playlist.AddAllToQueue(playlist.GetCurrentNode())
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(playlist)

				})

		case 'r':

			playlist.Refresh()

		}

		return e
	})

	return playlist

}

// Add songs and their directories in Playlist panel
func populate(root *tview.TreeNode, rootPath string) {

	files, err := ioutil.ReadDir(rootPath)

	if err != nil {
		appLog(err)
	}


	for _, file := range files {

		path := filepath.Join(rootPath, file.Name())
		f, err := os.Open(path)

		if err != nil {
			appLog(err)
			continue
		}

		defer f.Close()

		child := tview.NewTreeNode(file.Name())

		if !file.IsDir() {

			filetype, err := GetFileContentType(f)

			if err != nil {
				appLog(err)
				continue
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

			audioLength, err := GetLength(path)

			if err != nil {
				appLog(err)
				continue
			}

			audioFile := &AudioFile{
				Name:        file.Name(),
				Path:        path,
				IsAudioFile: true,
				Length:      audioLength,
				Parent:      root,
			}

			child.SetReference(audioFile)

		}


		if file.IsDir() {

			audioFile := &AudioFile{
				Name:        file.Name(),
				Path:        path,
				IsAudioFile: false,
				Length:      0,
				Parent:      root,
			}
			child.SetReference(audioFile)
			child.SetColor(gomu.AccentColor)
			populate(child, path)

		}

		// this is placed below because if
		// any of the checks above returns error
		// it will not be added to the playlist
		root.AddChild(child)

	}

}


// Bulk add a playlist to queue
func (p *Playlist) AddAllToQueue(root *tview.TreeNode) {

	var childrens []*tview.TreeNode

	childrens = root.GetChildren()

	// gets the parent if the highlighted item is a file
	if len(childrens) == 0 {
		childrens = root.GetReference().(*AudioFile).Parent.GetChildren()
	}

	for _, v := range childrens {
		currNode := v.GetReference().(*AudioFile)

		gomu.Queue.Enqueue(currNode)
	}

}

// Refresh the playlist and read the whole root music dir
func (p *Playlist) Refresh() {

	root := gomu.Playlist.GetRoot()

	prevFileName := gomu.Playlist.GetCurrentNode().GetText()

	root.ClearChildren()

	populate(root, root.GetReference().(*AudioFile).Path)

	root.Walk(func(node, parent *tview.TreeNode) bool {

		// to preserve previously highlighted node
		if node.GetReference().(*AudioFile).Name == prevFileName {
			p.SetHighlight(node)
			return false
		}

		return true
	})

}

// Adds child while setting reference to audio file
func (p *Playlist) AddSongToPlaylist(audioPath string, selPlaylist *tview.TreeNode) error {

	f, err := os.Open(audioPath)

	if err != nil {
		return err
	}

	defer f.Close()

	node := tview.NewTreeNode(path.Base(audioPath))

	audioLength, err := GetLength(audioPath)

	if err != nil {
		return err
	}

	audioFile := &AudioFile{
		Name:        path.Base(audioPath),
		Path:        audioPath,
		IsAudioFile: true,
		Length:      audioLength,
		Parent:      selPlaylist,
	}

	node.SetReference(audioFile)
	selPlaylist.AddChild(node)
	gomu.App.Draw()

	return nil

}

// Creates a directory under selected node, returns error if playlist exists
func (p *Playlist) CreatePlaylist(name string) error {

	selectedNode := p.GetCurrentNode()

	parentNode := selectedNode.GetReference().(*AudioFile).Parent

	// if the current node is the root
	// sets the parent to itself
	if parentNode == nil {
		parentNode = selectedNode
	}

	audioFile := parentNode.GetReference().(*AudioFile)

	err := os.Mkdir(path.Join(audioFile.Path, name), 0744)

	if err != nil {
		return err
	}

	p.Refresh()

	return nil

}

// This is used to replace default behaviour of SetCurrentNode which
// adds color highlight attributes
func (p *Playlist) SetHighlight(currNode *tview.TreeNode) {

	if p.prevNode != nil {
		p.prevNode.SetColor(gomu.TextColor)
	}
	currNode.SetColor(gomu.AccentColor)
	p.SetCurrentNode(currNode)

	if currNode.GetReference().(*AudioFile).IsAudioFile {
		p.prevNode = currNode
	}

}

// Traverses the playlist and finds the AudioFile struct
func (p *Playlist) FindAudioFile(audioName string) *AudioFile {

	root := p.GetRoot()

	if root == nil {
		return nil
	}

	var selNode *AudioFile

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		if audioFile.Name == audioName {
			selNode = audioFile
			return false
		}

		return true
	})

	return selNode

}


// download audio from youtube audio and adds the song to the selected playlist
func Ytdl(url string, selPlaylist *tview.TreeNode) {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		timedPopup(" Error ", "youtube-dl is not in your $PATH", getPopupTimeout())
		return
	}

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.Name

	timedPopup(" Ytdl ", "Downloading", getPopupTimeout())

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%s/%%(title)s.%%(ext)s",
		dir,
		selPlaylistName)

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {

		err := cmd.Run()
		if err != nil {
			timedPopup(" Error ", "Error running youtube-dl", getPopupTimeout())
			return
		}

		playlistPath := path.Join(expandTilde(dir), selPlaylistName)

		downloadedAudioPath := downloadedFilePath(
			stdout.Bytes(), playlistPath)

		err = gomu.Playlist.AddSongToPlaylist(downloadedAudioPath, selPlaylist)

		if err != nil {
			log.Println(err)
		}

		downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s", 
			path.Base(downloadedAudioPath))

		timedPopup(
			" Ytdl ",
			downloadFinishedMessage, 
			getPopupTimeout(),
		)

		gomu.App.Draw()

	}()

}
