package a

import (
	"testing"
)

// mocktail:Pineapple
// mocktail:Coconut

func TestName(t *testing.T) {
	var s Pineapple = newPineappleMock(t).
		OnHello(Water{}).TypedReturns("a").Once().
		OnWorld().TypedReturns("a").Once().
		OnGoo().TypedReturns("", 1, Water{}).Once().
		Parent

	s.Hello(Water{})
	s.World()
	s.Goo()
}
