package e

import (
	"a/c"
)

type V2Carrot interface {
	Bar(string) *V2Potato
	Bur(string) *c.Cherry
}

type V2Potato struct {
	Name string
}
