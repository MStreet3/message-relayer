package network

import "github.com/mstreet3/message-relayer/domain"

type NetworkSocket interface {
	Read() (domain.Message, error)
}
