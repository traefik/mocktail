package c_test

import (
	"context"
	"testing"

	"b/c"
)

// mocktail:Pineapple
// mocktail:Coconut

func TestName(t *testing.T) {
	var s c.Pineapple = c.NewPineappleMock(t).
		OnHello(c.Water{}).TypedReturns("a").Once().
		OnWorld().TypedReturns("a").Once().
		OnGoo().TypedReturns("", 1, c.Water{}).Once().
		OnCoo("", c.Water{}).TypedReturns(c.Water{}).
		TypedRun(func(s string, water c.Water) {}).Once().
		Parent

	s.Hello(c.Water{})
	s.World()
	s.Goo()
	s.Coo(context.Background(), "", c.Water{})

	fn := func(st c.Strawberry, stban c.Strawberry) c.Pineapple {
		return s
	}

	var coco c.Coconut = c.NewCoconutMock(t).
		OnLoo("a", 1, 2).TypedReturns("foo").Once().
		OnMoo(fn).TypedReturns("").Once().
		Parent

	coco.Loo("a", 1, 2)
	coco.Moo(fn)
}
