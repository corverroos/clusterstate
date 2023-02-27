package clusterstate

import (
	"encoding/json"
	"fmt"
	"math"
)

// Cluster represents a resulting cluster state at a given point in the DAG.
type Cluster struct {
	Height            int
	ApprovedMutations int
	Hashes            map[Hash]SignedMutation

	Name               string
	Operators          []Operator
	NumValidators      int
	WithdrawalAddress  string
	Validators         []Validator
	ParticipationProof []ParticipationProof
}

func (c Cluster) Clone() Cluster {
	b, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	var resp Cluster
	if err := json.Unmarshal(b, &resp); err != nil {
		panic(err)
	}

	return resp
}

// Operator represents a cluster operator, a human entity
// operating a charon node.
type Operator struct {
	PublicKey PublicKey
	ENR       string
}

// Resolve return the resulting cluster state at all heads/forks.
// It assumes that fork choice has already been applied to state.
func Resolve(state State) ([]Cluster, error) {
	if len(state) == 0 {
		return nil, fmt.Errorf("empty state")
	}

	var resp []Cluster
	for _, leaf := range state.Leaves() {
		sequence, err := state.Sequence(leaf)
		if err != nil {
			return nil, err
		}

		if sequence[0].Mutation.Type != TypeCreateCluster {
			return nil, fmt.Errorf("first mutation must be create cluster")
		}

		var cluster Cluster
		for _, mutation := range sequence {
			approvedBy, err := state.ApprovedBy(mutation.Hash)
			if err != nil {
				return nil, err
			}

			if !Approved(mutation.Mutation.Type.Approvals(), approvedBy, cluster) {
				break
			}

			cluster, err = AppendToCluster(mutation, cluster)
			if err != nil {
				return nil, err
			}
		}

		resp = append(resp, cluster)
	}

	return resp, nil
}

func Approved(require Approvals, approvedBy map[PublicKey]bool, cluster Cluster) bool {
	if require == ApprovalsNone {
		return true
	}

	var count int
	for _, op := range cluster.Operators {
		if approvedBy[op.PublicKey] {
			count++
		}
	}

	if require == ApprovalsQuorum {
		q := int(math.Ceil(float64(2*len(cluster.Operators)) / 3))
		return count >= q
	}

	return count == len(cluster.Operators)
}
