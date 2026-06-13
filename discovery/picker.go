package discovery

import (
	"math/rand/v2"
	"strings"
	"sync/atomic"
)

type Picker interface {
	Pick([]Endpoint) (Endpoint, error)
}

func NewPicker(policy string) Picker {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case LoadBalanceRandom:
		return &RandomPicker{}
	case LoadBalanceWeightedRound:
		return &WeightedRoundRobinPicker{}
	default:
		return &RoundRobinPicker{}
	}
}

type RoundRobinPicker struct {
	next atomic.Uint64
}

func (p *RoundRobinPicker) Pick(endpoints []Endpoint) (Endpoint, error) {
	if len(endpoints) == 0 {
		return Endpoint{}, ErrNoAvailableEndpoint
	}
	index := p.next.Add(1)
	return endpoints[int((index-1)%uint64(len(endpoints)))], nil
}

type RandomPicker struct{}

func (p *RandomPicker) Pick(endpoints []Endpoint) (Endpoint, error) {
	if len(endpoints) == 0 {
		return Endpoint{}, ErrNoAvailableEndpoint
	}
	return endpoints[rand.IntN(len(endpoints))], nil
}

type WeightedRoundRobinPicker struct {
	next atomic.Uint64
}

func (p *WeightedRoundRobinPicker) Pick(endpoints []Endpoint) (Endpoint, error) {
	if len(endpoints) == 0 {
		return Endpoint{}, ErrNoAvailableEndpoint
	}
	total := 0
	for _, endpoint := range endpoints {
		weight := endpoint.Weight
		if weight <= 0 {
			weight = 100
		}
		total += weight
	}
	if total <= 0 {
		return endpoints[0], nil
	}
	cursor := int(p.next.Add(1)%uint64(total)) + 1
	for _, endpoint := range endpoints {
		weight := endpoint.Weight
		if weight <= 0 {
			weight = 100
		}
		cursor -= weight
		if cursor <= 0 {
			return endpoint, nil
		}
	}
	return endpoints[len(endpoints)-1], nil
}
