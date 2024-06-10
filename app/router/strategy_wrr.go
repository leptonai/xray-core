package router

import (
	"context"
	"sync"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/extension"
)

// WeightedRoundRobinStrategy represents a random balancing strategy
type WeightedRoundRobinStrategy struct {
	ctx           context.Context
	observatory   extension.Observatory
	weightManager *WeightManager
	peers         map[string]*peer
	lock          sync.Mutex
}

type peer struct {
	tag             string
	weight          int
	currentWeight   int
	effectiveWeight int
}

func NewWeightedRoundRobinStrategy(settings *StrategyWeightedRoundRobinConfig) *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		weightManager: NewWeightManager(settings.Costs, 1, nil),
		peers:         make(map[string]*peer),
	}
}

func (s *WeightedRoundRobinStrategy) InjectContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *WeightedRoundRobinStrategy) GetPrincipleTarget(strings []string) []string {
	return strings
}

func (s *WeightedRoundRobinStrategy) PickOutbound(candidates []string) string {
	if s.observatory == nil {
		common.Must(core.RequireFeatures(s.ctx, func(observatory extension.Observatory) error {
			s.observatory = observatory
			return nil
		}))
	}

	peerTags := map[string]bool{}
	if s.observatory == nil {
		for _, candidate := range candidates {
			peerTags[candidate] = true
		}
	} else if observeReport, err := s.observatory.GetObservation(s.ctx); err == nil {
		if result, ok := observeReport.(*observatory.ObservationResult); ok {
			status := result.Status
			statusMap := make(map[string]*observatory.OutboundStatus)
			for _, outboundStatus := range status {
				statusMap[outboundStatus.OutboundTag] = outboundStatus
			}
			for _, candidate := range candidates {
				if outboundStatus, found := statusMap[candidate]; found {
					if outboundStatus.Alive {
						peerTags[candidate] = true
					}
				} else {
					// not found candidate is considered alive
					peerTags[candidate] = false
				}
			}
		}
	}

	if len(peerTags) == 0 {
		// goes to fallbackTag
		return ""
	}

	return s.selectPeer(peerTags)
}

func (s *WeightedRoundRobinStrategy) selectPeer(peerTags map[string]bool) string {
	s.lock.Lock()
	defer s.lock.Unlock()

	for tag := range s.peers {
		if _, ok := peerTags[tag]; !ok {
			delete(s.peers, tag)
		}
	}

	total := 0
	var best *peer
	for tag, alive := range peerTags {
		if !alive {
			continue
		}
		p := s.peers[tag]
		if p == nil {
			p = s.addPeer(tag)
		}

		p.currentWeight += p.effectiveWeight
		total += p.effectiveWeight
		//
		// NOTE: in the version, p.effectiveWeight is always less than p.weight.
		// So, the following condition is always true. But we keep it here for future use.
		// In Nginx, when the peer encounters any error, the effectiveWeight can be adjusted.
		// More details please refer to https://github.com/nginx/nginx/blob/e734df6664e70f118ca3140bcef6d4f1750fa8fa/src/stream/ngx_stream_upstream_round_robin.c#L642-L649
		//
		if p.effectiveWeight < p.weight {
			p.effectiveWeight++
		}

		if best == nil ||
			p.currentWeight > best.currentWeight {
			best = p
		}
	}

	if best == nil {
		// goes to fallbackTag
		return ""
	}

	best.currentWeight -= total
	return best.tag
}

func (s *WeightedRoundRobinStrategy) addPeer(tag string) *peer {
	weight := int(s.weightManager.Get(tag))
	if weight <= 0 {
		weight = 1
	}

	p := &peer{
		tag:             tag,
		weight:          weight,
		currentWeight:   0,
		effectiveWeight: weight,
	}
	s.peers[tag] = p
	return p
}
