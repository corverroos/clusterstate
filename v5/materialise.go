package v5

import "fmt"

func Materialise(dag RawDAG) (ClusterState, error) {
	var (
		state           ClusterState
		activeOperation                                    = OperationUnknown
		spread                                             = SpreadUnknown
		spreadValidator func(SignedMutation) (bool, error) = nil
	)
	for i, mutation := range dag {
		typ := mutation.Mutation.Type
		beginOperation, begins := typ.BeginsOperation()
		endsOperation := typ.EndsOperation()

		if activeOperation == OperationUnknown && !begins {
			return ClusterState{}, fmt.Errorf("mutation %d does not begin a new operation", i)
		} else if activeOperation != OperationUnknown && begins {
			return ClusterState{}, fmt.Errorf("mutation %d begins a new operation, but the previous operation has not ended", i)
		}

		if begins {
			activeOperation = beginOperation
		}

		if spread == SpreadUnknown {
			spread = typ.Spread()
			spreadValidator = spread.NewValidatorFunc(state)
		}

		spreadDone, err := spreadValidator(mutation)
		if err != nil {
			return ClusterState{}, fmt.Errorf("invalid mutation spread %d: %w", i, err)
		}

		state, err := typ.Transform(state, mutation)
		if err != nil {
			return ClusterState{}, fmt.Errorf("mutation transform %d: %w", i, err)
		}

		if spreadDone {
			spread = SpreadUnknown
			spreadValidator = nil
		}

	}
}

type OperationType string

const (
	OperationUnknown            OperationType = ""
	OperationProposeCluster     OperationType = "charon/operation/propose_cluster/1.0.0"
	OperationAcceptCluster      OperationType = "charon/operation/accept_cluster/1.0.0"
	OperationGenerateValidators OperationType = "charon/operation/generate_validators/1.0.0"
)

var operationDef = struct {
	structure
}{}
