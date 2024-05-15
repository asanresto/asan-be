package graph

import (
	"asan/graph/model"
	"sync"
)

//go:generate go run github.com/99designs/gqlgen generate

type Resolver struct {
	Channels sync.Map
}

func (r *Resolver) CreateChannel(key string) chan *model.Message {
	channel := make(chan *model.Message)
	r.Channels.Store(key, channel)
	return channel
}
