package clusterstate

import (
	"fmt"
	"sort"
)

func ValidateAdd(state State, sm SignedMutation) error {
	if len(state) == 0 && sm.Mutation.Type != TypeCreateCluster {
		return fmt.Errorf("first mutation must be create cluster")
	} else if sm.Mutation.Type.Approvals() == ApprovalsNone && len(sm.Mutation.ParentHashes) > 1 {
		return fmt.Errorf("approval mutation may only depend on a single parent")
	}

	allowedParents := sm.Mutation.Type.ParentTypes()

	clusters, err := Resolve(state)
	if err != nil {
		return err
	}
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].ApprovedMutations > clusters[j].ApprovedMutations
	})

	operators := make(map[PublicKey]bool)
	for _, operator := range clusters[0].Operators {
		operators[operator.PublicKey] = false
	}

	for _, p := range sm.Mutation.ParentHashes {
		parent, _, err := state.Get(p)
		if err != nil {
			return err
		}

		if !allowedParents[parent.Mutation.Type] {
			return fmt.Errorf("parent mutation type is not allowed")
		}

		if parent.Mutation.Type == sm.Mutation.Type && parent.Source == sm.Source {
			return fmt.Errorf("duplicate parent mutation")
		}

		if sm.Mutation.Type.Approvals() == ApprovalsNone {
			continue
		}

		if _, ok := clusters[0].Hashes[p]; !ok {
			return fmt.Errorf("parent mutation not in longest approved chain")
		}

		if operators[parent.Source] {
			return fmt.Errorf("duplicate parent source mutation")
		}
		operators[parent.Source] = true
	}

	return nil
}
