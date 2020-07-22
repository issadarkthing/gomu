package main

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rivo/tview"
)

// Prepares for test
func prepareTest() *Gomu {

	gomu := NewGomu()
	gomu.Player = &Player{}
	gomu.Queue = NewQueue()
	gomu.Playlist = &Playlist{
		tview.NewTreeView(),
		nil,
	}
	gomu.App = tview.NewApplication()

	rootDir, err := filepath.Abs("./test")
	if err != nil {
		panic(err)
	}

	root := tview.NewTreeNode("music")
	rootAudioFile := &AudioFile{
		Name: root.GetText(),
		Path: rootDir,
	}

	root.SetReference(rootAudioFile)
	populate(root, rootDir)
	gomu.Playlist.SetRoot(root)

	return gomu
}

func TestPopulate(t *testing.T) {

	gomu = NewGomu()
	rootDir, err := filepath.Abs("./test")

	if err != nil {
		panic(err)
	}

	expected := 0
	walkFn := func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			expected++
			return nil
		}

		f, e := os.Open(path)
		if e != nil {
			return e
		}

		defer f.Close()

		expected++

		return nil
	}

	// calculate the amount of mp3 files and directories
	filepath.Walk(rootDir, walkFn)

	root := tview.NewTreeNode(path.Base(rootDir))

	root.SetReference(&AudioFile{
		Name:        "Music",
		IsAudioFile: false,
	})

	populate(root, rootDir)
	gotItems := 1
	root.Walk(func(node, _ *tview.TreeNode) bool {
		gotItems++
		return true
	})

	if gotItems != expected {
		t.Errorf("Invalid amount of file; expected %d got %d", expected, gotItems)
	}

}

func TestAddAllToQueue(t *testing.T) {

	gomu = prepareTest()
	var songs []*tview.TreeNode

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		if node.GetReference().(*AudioFile).Name == "rap" {
			gomu.Playlist.AddAllToQueue(node)
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
