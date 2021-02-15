package anko_test

import (
	"testing"

	"github.com/issadarkthing/gomu/anko"
	"github.com/stretchr/testify/assert"
)

func TestDefine(t *testing.T) {
	a := anko.NewAnko()
	err := a.Define("x", 12)
	if err != nil {
		t.Error(err)
	}
}

func TestSet(t *testing.T) {
	a := anko.NewAnko()
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
	a := anko.NewAnko()

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
	a := anko.NewAnko()

	_, err := a.Execute(`x = 10`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetInt("x")
	assert.Equal(t, expect, got)

	a.Define("z", expect)
	val := a.GetInt("z")

	assert.Equal(t, expect, val)
}

func TestGetString(t *testing.T) {
	expect := "bruhh"
	a := anko.NewAnko()

	_, err := a.Execute(`x = "bruhh"`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetString("x")
	assert.Equal(t, expect, got)

	a.Define("z", expect)
	val := a.GetString("z")

	assert.Equal(t, expect, val)
}

func TestGetBool(t *testing.T) {
	expect := true
	a := anko.NewAnko()
	a.Define("x", expect)

	result := a.GetBool("x")
	assert.Equal(t, expect, result)
}

func TestExecute(t *testing.T) {
	expect := 12
	a := anko.NewAnko()

	_, err := a.Execute(`x = 6 + 6`)
	if err != nil {
		t.Error(err)
	}

	got := a.GetInt("x")
	assert.Equal(t, expect, got)
}
