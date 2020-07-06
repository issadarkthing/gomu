package main

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rivo/tview"
)

// Prepares for test
func preparePlaylist() *Gomu {

	gomu := NewGomu()
	gomu.Player = &Player{}
	gomu.Queue = NewQueue()
	gomu.Playlist = &Playlist{
		tview.NewTreeView(),
		nil,
	}
	gomu.App = tview.NewApplication()

	rootDir, err := filepath.Abs("./music")
	if err != nil {
		panic(err)
	}

	root := tview.NewTreeNode("music")
	rootAudioFile := &AudioFile{
		Name:        root.GetText(),
		Path:        rootDir,
		IsAudioFile: false,
		Parent:      nil,
	}

	root.SetReference(rootAudioFile)
	populate(root, rootDir)
	gomu.Playlist.SetRoot(root)

	return gomu
}

func TestPopulate(t *testing.T) {

	gomu = NewGomu()

	rootDir, err := filepath.Abs("./music")

	if err != nil {
		panic(err)
	}

	items := 0

	// calculate the amount of mp3 files
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			items++
			return nil
		}

		f, e := os.Open(path)
		if e != nil {
			return e
		}

		defer f.Close()

		fileType, e := GetFileContentType(f)

		if e != nil {
			return e
		}

		if fileType == "mpeg" {
			items++
		}

		return nil
	})

	root := tview.NewTreeNode(path.Base(rootDir))

	populate(root, rootDir)

	gotItems := 0
	root.Walk(func(node, parent *tview.TreeNode) bool {
		gotItems++
		return true
	})

	if gotItems != items {
		t.Error("populate() does not return correct amount of file")
	}

}



func TestAddAllToQueue(t *testing.T) {

	gomu = preparePlaylist()

	var songs []*tview.TreeNode

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		if node.GetReference().(*AudioFile).Name == "rap" {
			gomu.Playlist.AddAllToQueue(node)
			// remove first song because it will be popped right away
			songs = node.GetChildren()[1:]
		}

		return true
	})

	queue := gomu.Queue.GetItems()

	for i, song := range songs {

		audioFile := song.GetReference().(*AudioFile)

		// strips the path of the song in the queue
		s := filepath.Base(queue[i])

		if audioFile.Name != s {
			t.Errorf("Expected \"%s\", got \"%s\"", audioFile.Name, s)
		}

	}

}
