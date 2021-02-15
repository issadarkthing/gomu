package anko_test


import (
	"testing"

	"github.com/issadarkthing/gomu/anko"
	"github.com/stretchr/testify/assert"
)


func Test_GetBool(t *testing.T) {

	expect := true
	a := anko.NewAnko()
	a.Define("x", expect)

	result := a.GetBool("x")
	assert.Equal(t, expect, result)
}
