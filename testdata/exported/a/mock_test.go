package a

import (
	"context"
	"testing"
	"time"
)

// mocktail:Pineapple
// mocktail:Coconut
// mocktail:b.Carrot
// mocktail-:fmt.Stringer
// mocktail:Orange

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

	juiceCh := make(chan struct{}, 1)
	juiceCh <- struct{}{}

	var o Orange = NewOrangeMock(t).
		OnJuice().TypedReturns(juiceCh).Once().
		Parent

	select {
	case <-o.Juice():
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("timed out waiting for an orange juice")
	}
}
