package clusterstate

import (
	"fmt"
	"time"
)

type SimulateConfig struct {
	NumOperators int
}

// Simulate simulates te building of a cluster state.
func Simulate(conf SimulateConfig) error {
	creator := PublicKey("creator")
	operators := newOperators(conf.NumOperators)

	_ = State{newSignedMutation(
		creator,
		TypeCreateCluster,
		newCreateCluster(operators),
	)}

	panic("implement")
}

func newSignedMutation(souce PublicKey, typ MutationType, data any, parents ...SignedMutation) SignedMutation {
	m := newMutation(typ, data, parents...)

	return SignedMutation{
		Mutation: m,
		Hash:     m.Hash(),
		Source:   souce,
	}
}

func newMutation(typ MutationType, data any, parents ...SignedMutation) Mutation {
	var parentHashes []Hash
	for _, parent := range parents {
		parentHashes = append(parentHashes, parent.Hash)
	}

	return Mutation{
		ParentHashes: parentHashes,
		Type:         typ,
		Data:         data,
		Timestamp:    time.Now(),
	}
}

func newCreateCluster(operators []PublicKey) CreateCluster {
	return CreateCluster{
		Name:              "test-cluster",
		Operators:         operators,
		NumValidators:     1,
		WithdrawalAddress: "0x1234567890",
	}
}

func newOperators(n int) []PublicKey {
	var resp []PublicKey
	for i := 0; i < n; i++ {
		resp = append(resp, PublicKey(fmt.Sprintf("operator-%d", i)))
	}

	return resp
}
