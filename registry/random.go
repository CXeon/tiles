package registry

import (
	"math/rand"
)

type RandomBalancer struct {
}

func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{}
}

func (r *RandomBalancer) Select(eps []Endpoint) *Endpoint {
	length := len(eps)
	if length == 0 {
		return nil
	}
	ep := eps[rand.Intn(length)]
	return &ep
}
