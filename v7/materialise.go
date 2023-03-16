package v5

import "fmt"

func MaterialiseDV(dag RawDAG) (ClusterState, error) {
	var (
		state   ClusterState
		parent  Hash
		allowed = map[MutationType]bool{
			TypeCreateCluster:      true,
			TypeGenerateValidators: true,
			TypeAddValidators:      true,
		}
	)
	for i, mutation := range dag {
		if i == 0 {
			if mutation.Mutation.Type != TypeCreateCluster {
				return ClusterState{}, fmt.Errorf("first mutation must be TypeCreateCluster")
			}
		} else {
			if mutation.Mutation.Type == TypeCreateCluster {
				return ClusterState{}, fmt.Errorf("mutation %d is TypeCreateCluster", i)
			}
		}

		if !allowed[mutation.Mutation.Type] {
			return ClusterState{}, fmt.Errorf("mutation %d is not allowed", i)
		}

		if mutation.Mutation.Parent != parent {
			return ClusterState{}, fmt.Errorf("mutation %d has invalid parent", i)
		}

		var err error
		state, err = mutation.Mutation.Type.Transform(state, mutation)
		if err != nil {
			return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
		}

		parent = mutation.Hash
	}

	return state, nil
}
