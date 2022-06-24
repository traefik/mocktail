package b

import (
	"a/c"
)

type Carrot interface {
	Bar(string) *Potato
	Bur(string) *c.Cherry
}

type Potato struct {
	Name string
}
