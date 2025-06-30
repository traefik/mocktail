package a

import (
	"testing"
)

func TestGeneratedMocks(t *testing.T) {
	piniaColadaMock := newPiniaColadaMock(t).
		OnRhum().TypedReturns("rum").Once().
		OnPine("test.txt").Once().
		OnCoconut().Once().
		Parent

	piniaColadaMock.Rhum()
	piniaColadaMock.Pine("test.txt")
	piniaColadaMock.Coconut()
}
