package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/features/extension"
)

func TestWRR_selectPeer(t *testing.T) {
	tests := []struct {
		name             string
		candidateWeights []int
		times            int
		want             map[string]int
	}{
		{
			name:             "same weight",
			candidateWeights: []int{1, 1, 1, 1},
			times:            16,
			want: map[string]int{
				"tag-0": 4,
				"tag-1": 4,
				"tag-2": 4,
				"tag-3": 4,
			},
		},
		{
			name:             "diff weight",
			candidateWeights: []int{1, 2, 4, 8},
			times:            32,
			want: map[string]int{
				"tag-0": 2,
				"tag-1": 4,
				"tag-2": 9,
				"tag-3": 17,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var costs []*StrategyWeight
			peerTags := map[string]bool{}
			for i, v := range tt.candidateWeights {
				tag := fmt.Sprintf("tag-%d", i)
				costs = append(costs, &StrategyWeight{Match: tag, Value: float32(v)})
				peerTags[tag] = true
			}
			wrr := NewWeightedRoundRobinStrategy(&StrategyWeightedRoundRobinConfig{Costs: costs})
			var results []string
			counts := map[string]int{}
			for i := 0; i < tt.times; i++ {
				r := wrr.selectPeer(peerTags)
				counts[r]++
				results = append(results, r)
			}
			t.Log(results)
			assert.Equal(t, tt.want, counts)
		})
	}
}

type FakeObservatory struct {
	result observatory.ObservationResult
}

func (o *FakeObservatory) GetObservation(ctx context.Context) (proto.Message, error) {
	return &o.result, nil
}

func (o *FakeObservatory) Type() interface{} {
	return extension.ObservatoryType()
}

func (o *FakeObservatory) Start() error {
	return nil
}

func (o *FakeObservatory) Close() error {
	return nil
}

func TestWRR_Observatory(t *testing.T) {
	tests := []struct {
		name             string
		candidateWeights []int
		times            int
		unhealthyTags    map[string]struct{}
		want             map[string]int
	}{
		{
			name:             "diff weight",
			candidateWeights: []int{1, 2, 4, 8},
			unhealthyTags:    map[string]struct{}{"tag-2": {}},
			times:            32,
			want: map[string]int{
				"tag-0": 3,
				"tag-1": 6,
				// "tag-2": 9, unhealthy tag, should not be selected
				"tag-3": 23,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var costs []*StrategyWeight
			var peerTags []string
			observationResult := observatory.ObservationResult{}
			for i, v := range tt.candidateWeights {
				tag := fmt.Sprintf("tag-%d", i)
				costs = append(costs, &StrategyWeight{Match: tag, Value: float32(v)})
				peerTags = append(peerTags, tag)
				alive := true
				if _, ok := tt.unhealthyTags[tag]; ok {
					alive = false
				}
				observationResult.Status = append(observationResult.Status, &observatory.OutboundStatus{
					OutboundTag: tag,
					Alive:       alive,
				})
			}
			wrr := NewWeightedRoundRobinStrategy(&StrategyWeightedRoundRobinConfig{Costs: costs})
			wrr.observatory = &FakeObservatory{
				result: observationResult,
			}
			var results []string
			counts := map[string]int{}
			for i := 0; i < tt.times; i++ {
				r := wrr.PickOutbound(peerTags)
				counts[r]++
				results = append(results, r)
			}
			t.Log(results)
			assert.Equal(t, tt.want, counts)
		})
	}
}
