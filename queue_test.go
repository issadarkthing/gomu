package main

import (
	"testing"

	"github.com/rivo/tview"
)

var sample = map[string]string{
	"a": "1",
	"b": "2",
	"c": "3",
	"d": "4",
	"e": "5",
}

func TestQueueNext(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	q.SetCurrentItem(0)
	q.next()

	got := q.GetCurrentItem()

	if got != 1 {
		t.Errorf("Expected %d got %d", 1, got)
	}

}

func TestQueuePrev(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	q.SetCurrentItem(3)
	q.prev()

	got := q.GetCurrentItem()

	if got != 2 {
		t.Errorf("Expected %d got %d", 1, got)
	}

}

func TestQueueDeleteItem(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	initLen := q.GetItemCount()
	q.deleteItem(-1)
	finalLen := q.GetItemCount()

	if initLen != finalLen {
		t.Errorf("Item removed when -1 index was given")
	}

}

func TestQueuePop(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	initLen := q.GetItemCount()

	_, err := q.Dequeue()

	if err != nil {
		panic(err)
	}

	finalLen := q.GetItemCount()

	if finalLen == initLen {
		t.Errorf("Pop does not remove one element from the queue")
	}

	firstItem := q.GetItems()[0]

	got, _ := q.Dequeue()

	if got != firstItem {
		t.Errorf("Pop does not return the first item from the queue")
	}

}

func TestEnqueue(t *testing.T) {

	gomu = preparePlaylist()

	var audioFiles []*AudioFile

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		if len(audioFiles) < 2 && audioFile.IsAudioFile {
			audioFiles = append(audioFiles, audioFile)
			return false
		}

		return true
	})

	for _, v := range audioFiles {
		gomu.Queue.Enqueue(v)
	}

	queue := gomu.Queue.GetItems()

	// remove first item because it will be popped once enqueued
	for i, audioFile := range audioFiles[1:] {

		if queue[i] != audioFile.Path {
			t.Errorf("Invalid path; expected %s got %s", audioFile.Path, queue[i])
		}
	}

	queueLen := gomu.Queue.GetItemCount()

	if queueLen != 1 {
		t.Errorf("Invalid count in queue; expected %d, got %d", 1, queueLen)
	}

}


func TestQueueGetItems(t *testing.T) {

	q := NewQueue()

	for k, v := range sample {
		q.AddItem(k, v, 0, nil)
	}

	got := q.GetItems()

	if len(got) != len(sample) {
		t.Errorf("GetItems does not return correct items length")
	}

	sampleValues := []string{}

	for _, v := range sample {
		sampleValues = append(sampleValues, v)
	}

	for _, v := range got {
		if !SliceHas(v, sampleValues) {
			t.Error("GetItems does not return correct items")
		}
	}

}

// utility function to check elem in a slice
func SliceHas(item string, s []string) bool {

	for _, v := range s {
		if v == item {
			return true
		}
	}

	return false
}
