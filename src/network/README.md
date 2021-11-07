### network socket

the message relayer requires a network socket to `Read` messages from:

```go
type NetworkSocket interface {
	Read() (domain.Message, error)
}
```

this package implements a `NetworkSocketStub` that contains an
array of network responses for the message relayer to read.
