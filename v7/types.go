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

func (t MutationType) Transform(cl ClusterState, signedMutation SignedMutation) (ClusterState, error) {
	err := typeDef[t].VerifySigFunc(cl, signedMutation)
	if err != nil {
		return ClusterState{}, err
	}

	return typeDef[t].TransformFunc(cl, signedMutation)
}

// Validator represents a validator in the cluster.
type Validator struct {
	PublicKey    PublicKey
	PublicShares []PublicKey
}

// SignedMutation represents a mutation signed by the source that created it.
type SignedMutation struct {
	Mutation   Mutation
	Hash       Hash
	Source     PublicKey
	Signatures []byte
}

// Mutation represents a mutation to the cluster state, a vertex in the DAG.
type Mutation struct {
	Parent Hash
	Type   MutationType
	Data   any
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

type CreateCluster [2]SignedMutation

type OperatorENRs []SignedMutation

type GenerateValidators [2]SignedMutation

type Validators []Validator

type AddValidators [2]SignedMutation

type OperatorApprovals []SignedMutation
