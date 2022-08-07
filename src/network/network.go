package network

import "github.com/mstreet3/message-relayer/domain"

type NetworkReader interface {
	Read() (*domain.Message, error)
}

type Restarter interface {
	Restart() error
}

type RestartNetworkReader interface {
	Restarter
	NetworkReader
}
