package v5

import (
	"fmt"
)

const (
	TypeCreateCluster      MutationType = "charon/create_cluster/1.0.0"
	TypeProposeCluster     MutationType = "charon/propose_cluster/1.0.0"
	TypeOperatorENRs       MutationType = "charon/operator_enrs/1.0.0"
	TypeOperatorENR        MutationType = "charon/operator_enr/1.0.0"
	TypeGenerateValidators MutationType = "charon/generate_validators/1.0.0"
	TypeDKG                MutationType = "charon/dkg/1.0.0"
	TypeValidatorAck       MutationType = "charon/validator_ack/1.0.0"
	TypeAddValidators      MutationType = "charon/add_validators/1.0.0"
	TypeProposeValidators  MutationType = "charon/propose_validators/1.0.0"
	TypeOperatorApprovals  MutationType = "charon/operator_approvals/1.0.0"
	TypeOperatorApproval   MutationType = "charon/operator_approval/1.0.0"
)

var typeDef = map[MutationType]struct {
	DataType      any
	VerifySigFunc func(ClusterState, SignedMutation) error
	TransformFunc func(ClusterState, SignedMutation) (ClusterState, error)
}{
	TypeCreateCluster: {
		DataType: CreateCluster{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			cc, ok := mutation.Mutation.Data.(CreateCluster)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not CreateCluster")
			}

			if err := verifyLinearComposite(mutation, cc[:]); err != nil {
				return ClusterState{}, fmt.Errorf("invalid linear chain: %w", err)
			}

			if cc[0].Mutation.Type != TypeProposeCluster {
				return ClusterState{}, fmt.Errorf("first mutation is not ProposeCluster")
			} else if cc[1].Mutation.Type != TypeOperatorENRs {
				return ClusterState{}, fmt.Errorf("second mutation is not OperatorENRs")
			}

			for _, m := range cc {
				var err error
				state, err = m.Mutation.Type.Transform(state, m)
				if err != nil {
					return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
				}
			}

			return state, nil
		},
	},
	TypeProposeCluster: {
		DataType: ProposeCluster{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			// TODO(corver): validate signed mutation contains valid data
			// TODO(corver): validate state doesn't contain existing cluster

			state.Name = mutation.Mutation.Data.(ProposeCluster).Name

			state.Operators = make([]Operator, len(mutation.Mutation.Data.(ProposeCluster).Operators))
			for i := 0; i < len(state.Operators); i++ {
				state.Operators[i].PublicKey = mutation.Mutation.Data.(ProposeCluster).Operators[i]
			}

			state.Validators = make([]Validator, len(mutation.Mutation.Data.(ProposeCluster).Validators))
			// TODO(corver): Add validator feerecipient, withdrawal address, etc etc from to proposal to state.

			return state, nil
		},
	},
	TypeOperatorENRs: {
		DataType: OperatorENRs{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			enrs, ok := mutation.Mutation.Data.(OperatorENRs)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not OperatorENRs")
			}

			if err := verifyAllParallel(mutation, enrs[:], TypeOperatorENR, state.Operators); err != nil {
				return ClusterState{}, fmt.Errorf("invalid parralel composite: %w", err)
			}

			for _, m := range enrs {
				var err error
				state, err = m.Mutation.Type.Transform(state, m)
				if err != nil {
					return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
				}
			}

			return state, nil
		},
	},
	TypeOperatorENR: {
		DataType: OperatorENR{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			// TODO(corver): Verify valid ENR.

			enr := mutation.Mutation.Data.(OperatorENR).ENR

			for i := 0; i < len(state.Operators); i++ {
				if state.Operators[i].PublicKey == mutation.Source {
					if state.Operators[i].ENR != "" {
						return state, fmt.Errorf("operator already has enr")
					}

					state.Operators[i].ENR = enr

					return state, nil
				}
			}

			return state, fmt.Errorf("operator not found")
		},
	},
	TypeGenerateValidators: {
		DataType: GenerateValidators{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			gv, ok := mutation.Mutation.Data.(GenerateValidators)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not GenerateValidators")
			}

			if err := verifyLinearComposite(mutation, gv[:]); err != nil {
				return ClusterState{}, fmt.Errorf("invalid linear composite: %w", err)
			}

			if gv[0].Mutation.Type != TypeDKG {
				return ClusterState{}, fmt.Errorf("first mutation is not DKG")
			} else if gv[1].Mutation.Type != TypeValidatorAck {
				return ClusterState{}, fmt.Errorf("second mutation is not ValidatorAck")
			}

			for _, m := range gv {
				var err error
				state, err = m.Mutation.Type.Transform(state, m)
				if err != nil {
					return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
				}
			}

			return state, nil
		},
	},
	TypeDKG: {
		DataType: Validators{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			vals, ok := mutation.Mutation.Data.(Validators)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not DKG")
			}

			if len(state.Validators) != len(vals) {
				return ClusterState{}, fmt.Errorf("number of validators does not match number of DKGs")
			}

			state.Validators = vals

			return state, nil
		},
	},
	TypeValidatorAck: {
		DataType: nil,
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			return state, nil
		},
	},
	TypeAddValidators: {
		DataType: AddValidators{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			av, ok := mutation.Mutation.Data.(AddValidators)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not AddValidators")
			}

			if err := verifyLinearComposite(mutation, av[:]); err != nil {
				return ClusterState{}, fmt.Errorf("invalid linear composite: %w", err)
			}

			if av[0].Mutation.Type != TypeProposeValidators {
				return ClusterState{}, fmt.Errorf("first mutation is not ProposeValidators")
			} else if av[1].Mutation.Type != TypeOperatorApprovals {
				return ClusterState{}, fmt.Errorf("second mutation is not OperatorApprovals")
			}

			for _, m := range av {
				var err error
				state, err = m.Mutation.Type.Transform(state, m)
				if err != nil {
					return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
				}
			}

			return state, nil
		},
	},
	TypeProposeValidators: {
		DataType: Validators{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			vals, ok := mutation.Mutation.Data.(Validators)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not Validators")
			}
			state.Validators = append(state.Validators, vals...)

			return state, nil
		},
	},
	TypeOperatorApprovals: {
		DataType: OperatorApprovals{},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			approvals, ok := mutation.Mutation.Data.(OperatorApprovals)
			if !ok {
				return ClusterState{}, fmt.Errorf("mutation data is not OperatorApprovals")
			}

			if err := verifyAllParallel(mutation, approvals[:], TypeOperatorApproval, state.Operators); err != nil {
				return ClusterState{}, fmt.Errorf("invalid parralel composite: %w", err)
			}

			for _, m := range approvals {
				var err error
				state, err = m.Mutation.Type.Transform(state, m)
				if err != nil {
					return ClusterState{}, fmt.Errorf("transform mutation: %w", err)
				}
			}

			return state, nil
		},
	},
	TypeOperatorApproval: {
		DataType: nil,
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
			return state, nil
		},
	},
}

func verifyAllParallel(parent SignedMutation, mutations []SignedMutation, expectedType MutationType, operators []Operator) error {
	if len(mutations) != len(operators) {
		return fmt.Errorf("number of parralel mutations do not match number of operators")
	}

	for i, m := range mutations {
		if m.Source != operators[i].PublicKey {
			return fmt.Errorf("mutation %d has invalid source", i)
		}

		if m.Mutation.Type != expectedType {
			return fmt.Errorf("mutation %d has invalid type", i)
		}
		if m.Mutation.Parent != parent.Hash {
			return fmt.Errorf("mutation %d has invalid parent", i)
		}
	}

	return nil
}

func verifyLinearComposite(parent SignedMutation, mutations []SignedMutation) error {
	for i, m := range mutations {
		if m.Mutation.Parent != parent.Hash {
			return fmt.Errorf("mutation %d has invalid parent", i)
		}
	}

	return nil
}
