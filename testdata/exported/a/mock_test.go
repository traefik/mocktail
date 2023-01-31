package a_test

import (
	"context"
	"testing"
	"time"

	"a"
)

// mocktail:Pineapple
// mocktail:Coconut
// mocktail:b.Carrot
// mocktail-:fmt.Stringer
// mocktail:Orange

func TestName(t *testing.T) {
	var s a.Pineapple = a.NewPineappleMock(t).
		OnHello(a.Water{}).TypedReturns("a").Once().
		OnWorld().TypedReturns("a").Once().
		OnGoo().TypedReturns("", 1, a.Water{}).Once().
		OnCoo("", a.Water{}).TypedReturns(a.Water{}).
		TypedRun(func(s string, water a.Water) {}).Once().
		Parent

	s.Hello(a.Water{})
	s.World()
	s.Goo()
	s.Coo(context.Background(), "", a.Water{})

	fn := func(st a.Strawberry, stban a.Strawberry) a.Pineapple {
		return s
	}

	var c a.Coconut = a.NewCoconutMock(t).
		OnLoo("a", 1, 2).TypedReturns("foo").Once().
		OnMoo(fn).TypedReturns("").Once().
		Parent

	c.Loo("a", 1, 2)
	c.Moo(fn)

	juiceCh := make(chan struct{}, 1)
	juiceCh <- struct{}{}

	var o a.Orange = a.NewOrangeMock(t).
		OnJuice().TypedReturns(juiceCh).Once().
		Parent

	select {
	case <-o.Juice():
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("timed out waiting for an orange juice")
	}
}
