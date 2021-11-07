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
