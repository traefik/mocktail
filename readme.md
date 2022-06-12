# Mocktail

Naive code generator that create mock implementation using `testify.mock`.

It requires testify >= v1.7.0

Unlike [mockery](https://github.com/vektra/mockery), Mocktail generates typed methods on mocks.

How to use:
- Create a file named `mock_test.go` inside the package that you can to create mocks.
- Add one or multiple comments `// mocktail:MyInterface` inside the file `mock_test.go`.

```go
package example

// mocktail:MyInterface

```

Replacement pattern:
```
([.\s]On)\("([^"]+)",?

$1$2(
```
