package c

import (
	"context"
	"testing"
)

// mocktail:Pineapple
// mocktail:Coconut

func TestName(t *testing.T) {
	var s Pineapple = NewPineappleMock(t).
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

	fn := func(st Strawberry, stban Strawberry) Pineapple {
		return s
	}

	var c Coconut = NewCoconutMock(t).
		OnLoo("a", 1, 2).TypedReturns("foo").Once().
		OnMoo(fn).TypedReturns("").Once().
		Parent

	c.Loo("a", 1, 2)
	c.Moo(fn)
}
