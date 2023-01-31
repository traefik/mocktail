# Mocktail

<p align="center">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="./mocktail-dark.png">
      <source media="(prefers-color-scheme: light)" srcset="./mocktail.png">
      <img alt="Mocktail logo" src="./mocktail.png">
    </picture>
</p>

Naive code generator that creates mock implementation using `testify.mock`.

Unlike [mockery](https://github.com/vektra/mockery), Mocktail generates typed methods on mocks.

## How to use

- Create a file named `mock_test.go` inside the package that you can to create mocks.
- Add one or multiple comments `// mocktail:MyInterface` inside the file `mock_test.go`.

```go
package example

// mocktail:MyInterface

```

## How to Install

### Go Install

You can install Mocktail by running the following command:

```bash
go install github.com/traefik/mocktail@latest
```

### Pre-build Binaries

You can use pre-compiled binaries:

* To get the binary just download the latest release for your OS/Arch from [the releases page](https://github.com/traefik/mocktail/releases)
* Unzip the archive.
* Add `mocktail` in your `PATH`.

## Notes

It requires testify >= v1.7.0

Mocktail can only generate mock of interfaces inside a module itself (not from stdlib or dependencies)

The `// mocktail` comments **must** be added to a file named `mock_test.go` only,  
comments in other files will not be detected

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

## Exportable Mocks

If you need to use your mocks in external packages just add flag `-e`:

```shell
mocktail -e
```

In this case, mock will be created in the same package but in the file `mock_gen.go`.

<!--

Replacement pattern:
```
([.\s]On)\("([^"]+)",?

$1$2(
```

-->
