package anko

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestDefine(t *testing.T) {
	a := NewAnko()
	err := a.Define("x", 12)
	if err != nil {
		t.Error(err)
	}
}

func TestSet(t *testing.T) {
	a := NewAnko()
	err := a.Define("x", 12)
	if err != nil {
		t.Error(err)
	}

	err = a.Set("x", 12)
	if err != nil {
		t.Error(err)
	}
}

func TestGet(t *testing.T) {
	a := NewAnko()

	expect := 12
	err := a.Define("x", expect)
	if err != nil {
		t.Error(err)
	}

	got, err := a.Get("x")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, expect, got)

	if err != nil {
		t.Error(err)
	}
}

func TestGetInt(t *testing.T) {
	expect := 10
	a := NewAnko()

	_, err := a.Execute(`x = 10`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetInt("x")
	assert.Equal(t, expect, got)

	_, err = a.Execute(`module S { x = 10 }`)
	if err != nil {
		t.Error(err)
	}

	got = a.GetInt("S.x")
	assert.Equal(t, expect, got)

	got = a.GetInt("S.y")
	assert.Equal(t, 0, got)

	a.Define("z", expect)
	val := a.GetInt("z")

	assert.Equal(t, expect, val)
}

func TestGetString(t *testing.T) {
	expect := "bruhh"
	a := NewAnko()

	_, err := a.Execute(`x = "bruhh"`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetString("x")
	assert.Equal(t, expect, got)

	_, err = a.Execute(`module S { x = "bruhh" }`)
	if err != nil {
		t.Error(err)
	}

	got = a.GetString("S.x")
	assert.Equal(t, expect, got)

	got = a.GetString("S.y")
	assert.Equal(t, "", got)

	a.Define("z", expect)
	val := a.GetString("z")

	assert.Equal(t, expect, val)
}

func TestGetBool(t *testing.T) {
	expect := true
	a := NewAnko()
	a.Define("x", expect)

	_, err := a.Execute(`module S { x = true }`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetBool("S.x")
	assert.Equal(t, expect, got)

	got = a.GetBool("S.y")
	assert.Equal(t, false, got)

	result := a.GetBool("x")
	assert.Equal(t, expect, result)
}

func TestExecute(t *testing.T) {
	expect := 12
	a := NewAnko()

	_, err := a.Execute(`x = 6 + 6`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetInt("x")
	assert.Equal(t, expect, got)
}

func TestExtractCtrlRune(t *testing.T) {
	tests := []struct {
		in  string
		out rune
	}{
		{in: "Ctrl+x", out: 'x'},
		{in: "Ctrl+]", out: ']'},
		{in: "Ctrl+%", out: '%'},
		{in: "Ctrl+^", out: '^'},
		{in: "Ctrl+7", out: '7'},
		{in: "Ctrl+B", out: 'B'},
	}

	for _, test := range tests {
		got := extractCtrlRune(test.in)
		assert.Equal(t, test.out, got)
	}
}

func TestExtractAltRune(t *testing.T) {
	tests := []struct {
		in  string
		out rune
	}{
		{in: "Alt+Rune[x]", out: 'x'},
		{in: "Alt+Rune[]]", out: ']'},
		{in: "Alt+Rune[%]", out: '%'},
		{in: "Alt+Rune[^]", out: '^'},
		{in: "Alt+Rune[7]", out: '7'},
		{in: "Alt+Rune[B]", out: 'B'},
	}

	for _, test := range tests {
		got := extractAltRune(test.in)
		assert.Equal(t, test.out, got)
	}
}

func TestKeybindExists(t *testing.T) {

	tests := []struct {
		panel  string
		key    *tcell.EventKey
		exists bool
	}{
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone),
			exists: true,
		},
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone),
			exists: false,
		},
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyRune, ']', tcell.ModNone),
			exists: true,
		},
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyRune, '[', tcell.ModNone),
			exists: false,
		},
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyCtrlB, 'b', tcell.ModCtrl),
			exists: true,
		},
		{
			panel:  "global",
			key:    tcell.NewEventKey(tcell.KeyCtrlC, 'c', tcell.ModCtrl),
			exists: false,
		},
		{
			panel:  "playlist",
			key:    tcell.NewEventKey(tcell.KeyRune, '!', tcell.ModAlt),
			exists: true,
		},
		{
			panel:  "playlist",
			key:    tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModAlt),
			exists: false,
		},
		{
			panel:  "playlist",
			key:    tcell.NewEventKey(tcell.KeyCtrlCarat, '^', tcell.ModCtrl),
			exists: true,
		},
		{
			panel:  "queue",
			key:    tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModAlt),
			exists: true,
		},
		{
			panel:  "queue",
			key:    tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
			exists: true,
		},
	}

	src := `
module Keybinds {
	global = {}
	playlist = {}
	queue = {}

	global["b"] = func() { return 0 }
	global["]"] = func() { return 0 }
	global["ctrl_b"] = func() { return 0 }
	global["alt_b"] = func() { return 0 }

	playlist["alt_!"] = func() { return 0 }
	playlist["ctrl_^"] = func() { return 0 }

	queue["alt_>"] = func() { return 0 }
	queue["enter"] = func() { return 0 }
}
`
	a := NewAnko()

	_, err := a.Execute(src)
	if err != nil {
		t.Error(err)
	}

	for i, test := range tests {
		got := a.KeybindExists(test.panel, test.key)
		msg := fmt.Sprintf("error on test %d", i+1)
		assert.Equal(t, test.exists, got, msg)
	}
}
