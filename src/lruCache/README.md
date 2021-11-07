### least recently used (LIFO) queue

this package implements a simple last in first out (LIFO)
queue for caching network messages. The queue implements the interface:

```go
type PriorityQueue interface {
	Pop() (*domain.Message, bool)
	Push(domain.Message)
}
```

tests of the queue are run via:

```bash
> go test priorityQueue_test.go
```
