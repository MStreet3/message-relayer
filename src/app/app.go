package app

import (
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
)

func Start() {
	responses := []network.NetworkResponse{
		{Message: domain.Message{Type: domain.ReceivedAnswer}}}
	ns := network.NewNetworkSocketStub(responses)
	mr := relayer.NewMessageRelayer(ns)
	ch := make(chan domain.Message)
	mr.SubscribeToMessage(domain.ReceivedAnswer, ch)
	mr.Listen()
}
