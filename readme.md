# Mocktail

![](./mocktail.png)

Naive code generator that create mock implementation using `testify.mock`.

Unlike [mockery](https://github.com/vektra/mockery), Mocktail generates typed methods on mocks.

## How to use

- Create a file named `mock_test.go` inside the package that you can to create mocks.
- Add one or multiple comments `// mocktail:MyInterface` inside the file `mock_test.go`.

```go
package example

// mocktail:MyInterface

```

## Notes

It requires testify >= v1.7.0

Mocktail can only generate mock of interfaces inside a module itself (not from stdlib or dependencies)

## Examples

```go
package a

import (
	"context"
	"time"
)

type Pineapple interface {
	Juice(context.Context, string, Water) Water
}

type Coconut interface {
	Open(string, int) time.Duration
}

type Water struct{}
```

```go
package a

import (
	"context"
	"testing"
)

// mocktail:Pineapple
// mocktail:Coconut

func TestMock(t *testing.T) {
	var s Pineapple = newPineappleMock(t).
		OnJuice("foo", Water{}).TypedReturns(Water{}).Once().
		Parent

	s.Juice(context.Background(), "", Water{})

	var c Coconut = newCoconutMock(t).
		OnOpen("bar", 2).Once().
		Parent

	c.Open("a", 2)
}
```

<!--

Replacement pattern:
```
([.\s]On)\("([^"]+)",?

$1$2(
```

-->
