package a

import (
	"testing"
)

func TestGeneratedMocks(t *testing.T) {
	var piniaColadaMock PiniaColada = newPiniaColadaMock(t).
		OnRhum().TypedReturns("rum").Once().
		OnPine("test.txt").Once().
		OnCoconut().Once().
		Parent

	piniaColadaMock.Rhum()
	piniaColadaMock.Pine("test.txt")
	piniaColadaMock.Coconut()

	var shirleyTempleMock shirleyTemple = newShirleyTempleMock(t).
		Onale("test.txt").Once().
		OnGrenadine().Once().
		OnGetCherry().TypedReturns("maraschino").Once().
		Parent

	shirleyTempleMock.ale("test.txt")
	shirleyTempleMock.Grenadine()
	shirleyTempleMock.GetCherry()
}
