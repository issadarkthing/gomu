package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestGetFn(t *testing.T) {

	c := newCommand()

	c.define("sample", func() {})

	f, err := c.getFn("sample")
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, f)

	f, err = c.getFn("x")
	assert.Error(t, err)
}
