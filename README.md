### message relayer

this repo implements a `message relayer` and a
`message relayer server` with the interfaces:

```go
type MessageRelayer interface {
	SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message)
}

type MessageRelayerServer interface {
	MessageRelayer
	ReadAndRelay() // serve
}
```

the default implementation of the `MessageRelayerServer` relays
messages to its subscribers as messages are read from the network.

`app.go` provides a sample use of the api. in the example a relayer
server runs until it's network crashes (simulated by an empty set of
responses).

```bash
> cd ./src
> go run main.go
```

there are test cases agains the `DefaultMessageRelayer` which can be run
via:

```bash
> cd ./src/relayer
> go test -run "_default"
```

by default `debug` logging is off. debug out put can be included by flipping the flag
in `./src/utils/utils.go`.

### priority message relayer

the `priority message relayer` limits the number of messages that it keeps queued locally to avoid runaway memory consumption, but the protocol imposes some rules about the priority of these messages:

- We must always ensure that we broadcast the 2 most recent StartNewRound messages
- We only need to ensure that we broadcast only the most recent ReceivedAnswer message
- Any time that both a “StartNewRound” and a “ReceivedAnswer” are queued, the “StartNewRound” message should be broadcasted first
- Any time that one of the subscribers of this system is busy and cannot receive a message immediately, we should just skip broadcasting to that subscriber

the `priority message relayer` achieves these goals by first `Enqueue`ing each message from the network
onto a priority queue and then `Dequeue`ing and `Broadcast`ing the messages from the queue.

there are test cases agains the `PriorityMessageRelayer` which can be run
via:

```bash
> cd ./src/relayer
> go test -run "_priority"
```
