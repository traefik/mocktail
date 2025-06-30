package b

import (
	"context"
	"testing"
)

func TestGeneratedMocks(t *testing.T) {
	piniaColadaMock := NewPiniaColadaMock(t).
		OnRhum().TypedReturns("rum").Once().
		OnPine("test.txt").Once().
		OnCoconut().Once().
		Parent

	piniaColadaMock.Rhum(context.Background())
	piniaColadaMock.Pine("test.txt")
	piniaColadaMock.Coconut()
}
