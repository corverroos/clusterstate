package v5

import (
	"crypto/sha256"
	"encoding/json"
)

// DutyType represents the type of a validator duty; attester, proposer, etc.
type DutyType int

// PublicKey represents a public key, either of a charon node or an operator or a validator.
type PublicKey string

// Hash represents a 32 byte hash.
type Hash [32]byte

// MutationType represents the type of a mutation.
type MutationType string

func (t MutationType) BeginsOperation() (OperationType, bool) {
	return typeDef[t].BeginsOperation, typeDef[t].BeginsOperation != OperationUnknown
}
func (t MutationType) EndsOperation() bool {
	return typeDef[t].EndsOperation
}

func (t MutationType) Transform(cl ClusterState, signedMutation SignedMutation) (ClusterState, error) {
	return typeDef[t].TransformFunc(cl, signedMutation)
}

func (t MutationType) Spread() Spread {
	return typeDef[t].Spread
}

// Validator represents a validator in the cluster.
type Validator struct {
	PublicKey    PublicKey
	PublicShares []PublicKey
}

const (
	TypeProposeCluster     MutationType = "charon/propose_cluster/1.0.0"
	TypeAcceptClusterBegin MutationType = "charon/accept_cluster_begin/1.0.0"
	TypeAcceptClusterEnd   MutationType = "charon/accept_cluster_end/1.0.0"
	TypeOperatorENR        MutationType = "charon/operator_enr/1.0.0"
	TypeDKG                MutationType = "charon/dkg/1.0.0"
)

// SignedMutation represents a mutation signed by the source that created it.
type SignedMutation struct {
	Mutation   Mutation
	Hash       Hash
	Source     PublicKey
	Signatures []byte
}

// Mutation represents a mutation to the cluster state, a vertex in the DAG.
type Mutation struct {
	ParentMutationHashes []Hash
	ParentOperationHash  Hash
	Type                 MutationType
	Data                 any
}

func (m Mutation) Hash() Hash {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return Hash(sha256.Sum256(b))
}

type ProposeCluster struct {
	Name       string
	Operators  []PublicKey
	Validators []Validator
}

type OperatorENR struct {
	ENR string
}

type RawDAG []SignedMutation
