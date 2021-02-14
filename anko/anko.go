package anko

import (
	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	"github.com/mattn/anko/vm"
	_ "github.com/mattn/anko/packages"
)


type Anko struct {
	env *env.Env
}

func NewAnko() Anko {
	return Anko{
		core.Import(env.NewEnv()),
	}
}

// define defines new symbol and value to the Anko env
func (a *Anko) Define(symbol string, value interface{}) error {
	return a.env.DefineGlobal(symbol, value)
}

// set sets new value to existing symbol. Use this when change value under an
// existing symbol.
func (a *Anko) Set(symbol string, value interface{}) error {
	return a.env.Set(symbol, value)
}

// get gets value from anko env, returns error if symbol is not found.
func (a *Anko) Get(symbol string) (interface{}, error) {
	return a.env.Get(symbol)
}

// getInt gets int value from symbol, returns golang default value if not found
func (a *Anko) GetInt(symbol string) int {
	v, err := a.env.Get(symbol)
	if err != nil {
		return 0
	}

	val, ok := v.(int64)
	if !ok {
		return 0
	}

	return int(val)
}

// getString gets string value from symbol, returns golang default value if not
// found
func (a *Anko) GetString(symbol string) string {
	v, err := a.env.Get(symbol)
	if err != nil {
		return ""
	}

	val, ok := v.(string)
	if !ok {
		return ""
	}

	return val
}

// getBool gets bool value from symbol, returns golang default value if not
// found
func (a *Anko) GetBool(symbol string) bool {
	v, err := a.env.Get(symbol)
	if err != nil {
		return false
	}

	val, ok := v.(bool)
	if !ok {
		return false
	}

	return val
}

// execute executes anko script
func (a *Anko) Execute(src string) (interface{}, error) {
	return vm.Execute(a.env, nil, src)
}
