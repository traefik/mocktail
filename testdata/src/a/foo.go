package a

import (
	"bytes"
	"time"

	"golang.org/x/mod/module"
)

type Pineapple interface {
	Hello(bar Water) string
	World() string
	Goo() (string, int, Water)
}

type Coconut interface {
	Hoo(string, int, Water)
	Joo(string, int, Water) (string, int)
	Koo(src string) (dst string)
	Too(src string) time.Duration
	Doo(src time.Duration) time.Duration
	Boo(src *bytes.Buffer) time.Duration
	Voo(src *module.Version) time.Duration
}

type Water struct{}
