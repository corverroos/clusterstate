package v5

import "fmt"

var typeDef = map[MutationType]struct {
	DataType         any
	ParentOperations []OperationType

	BeginsOperation OperationType
	EndsOperation   bool

	Spread Spread

	ValidateFunc  func(ClusterState, SignedMutation) error
	TransformFunc func(ClusterState, SignedMutation) (ClusterState, error)
}{
	TypeProposeCluster: {
		BeginsOperation: OperationAcceptCluster,
		EndsOperation:   true,
		ValidateFunc: func(state ClusterState, mutation SignedMutation) error {
			// TODO(corver): validate signed mutation contains valid data
			// TODO(corver): validate state doesn't contain existing cluster

			return nil
		},
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
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
	TypeAcceptClusterBegin: {
		BeginsOperation:  OperationAcceptCluster,
		EndsOperation:    false,
		ParentOperations: []OperationType{OperationAcceptCluster},
	},
	TypeOperatorENR: {
		BeginsOperation: OperationAcceptCluster,
		EndsOperation:   true,
		Spread:          SpreadOperatorsAll,
		TransformFunc: func(state ClusterState, mutation SignedMutation) (ClusterState, error) {
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
	TypeAcceptClusterEnd: {
		EndsOperation: true,
	},
	TypeDKG: {
		BeginsOperation: OperationGenerateValidators,
		EndsOperation:   true,
	},
}

type Spread int

func (s Spread) NewValidatorFunc(state ClusterState) func(SignedMutation) (bool, error) {
	return spreadDef[s].NewValidatorFunc(state)
}

const (
	SpreadUnknown Spread = iota
	SpreadOne
	SpreadOperatorsAll
	SpreadValidatorsAll
)

var spreadDef = map[Spread]struct {
	NewValidatorFunc func(ClusterState) func(SignedMutation) (bool, error)
}{
	SpreadOne: {
		NewValidatorFunc: func(ClusterState) func(SignedMutation) (bool, error) {
			var count int
			return func(mutation SignedMutation) (bool, error) {
				count++
				if count > 1 {
					return false, fmt.Errorf("too many mutations")
				}
				return true, nil
			}
		},
	},
	SpreadOperatorsAll: {
		NewValidatorFunc: func(state ClusterState) func(SignedMutation) (bool, error) {
			var idx int
			return func(mutation SignedMutation) (bool, error) {
				if idx > len(state.Operators) {
					return false, fmt.Errorf("too many mutations")
				}

				if state.Operators[idx].PublicKey != mutation.Source {
					return false, fmt.Errorf("mutation from wrong operator")
				}

				idx++

				return idx == len(state.Operators), nil
			}
		},
	},
}
