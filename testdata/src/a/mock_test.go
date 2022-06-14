package a

import (
	"context"
	"testing"
)

// mocktail:Pineapple
// mocktail:Coconut

func TestName(t *testing.T) {
	var s Pineapple = newPineappleMock(t).
		OnHello(Water{}).TypedReturns("a").Once().
		OnWorld().TypedReturns("a").Once().
		OnGoo().TypedReturns("", 1, Water{}).Once().
		OnCoo("", Water{}).TypedReturns(Water{}).
		TypedRun(func(s string, water Water) {}).Once().
		Parent

	s.Hello(Water{})
	s.World()
	s.Goo()
	s.Coo(context.Background(), "", Water{})

	var c Coconut = newCoconutMock(t).
		OnLoo("a", 1, 2).TypedReturns("foo").Once().
		Parent

	c.Loo("a", 1, 2)
}
