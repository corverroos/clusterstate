package clusterstate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// DutyType represents the type of a validator duty; attester, proposer, etc.
type DutyType int

// PublicKey represents a public key, either of a charon node or an operator or a validator.
type PublicKey string

// Hash represents a 32 byte hash.
type Hash [32]byte

// MutationType represents the type of a mutation.
type MutationType string

func (t MutationType) Approvals() Approvals {
	return typeDef[t].Approvals
}

func (t MutationType) ParentTypes() map[MutationType]bool {
	resp := make(map[MutationType]bool)
	for _, p := range typeDef[t].ParentTypes {
		resp[p] = true
	}

	return resp
}

const (
	TypeCreateCluster      MutationType = "charon/create_cluster/1.0.0"
	TypeOperatorENR        MutationType = "charon/operator_enr/1.0.0"
	TypeGenerateValidators MutationType = "charon/generate_validators/1.0.0"
	TypeAddValidators      MutationType = "charon/add_validators/1.0.0"
	TypeOperatorAck        MutationType = "charon/operator_ack/1.0.0"
	TypeChangeOperators    MutationType = "charon/change_operators/1.0.0"
	TypeReshareValidators  MutationType = "charon/reshare_validators/1.0.0"
	TypeParticipationProof MutationType = "charon/participation_proof/1.0.0"
)

// SignedMutation represents a mutation signed by the source that created it.
type SignedMutation struct {
	Mutation  Mutation
	Hash      Hash
	Source    PublicKey
	Signature []byte
}

// Mutation represents a mutation to the cluster state, a vertex in the DAG.
type Mutation struct {
	ParentHashes []Hash
	Type         MutationType
	Data         any
	Timestamp    time.Time
}

func (m Mutation) Hash() Hash {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return Hash(sha256.Sum256(b))
}

// CreateCluster represents the TypeCreateCluster mutation data.
type CreateCluster struct {
	Name              string
	Operators         []PublicKey
	NumValidators     int
	WithdrawalAddress string
}

// OperatorENR represents the TypeOperatorENR mutation data.
type OperatorENR struct {
	ENR string
}

// GenerateValidators represents the TypeGenerateValidators mutation data.
type GenerateValidators struct {
	Validators []Validator
}

// Validator represents a validator in the cluster.
type Validator struct {
	PublicKey    PublicKey
	PublicShares []PublicKey
}

// AddValidators represents the TypeAddValidators mutation data.
type AddValidators struct {
	NumValidators int
}

// OperatorAck represents the TypeOperatorAck (noop) mutation data.
type OperatorAck struct{}

// ChangeOperators represents the TypeChangeOperators mutation data.
type ChangeOperators struct {
	NewOperators []PublicKey
}

// ReshareValidators represents the TypeReshareValidators mutation data.
type ReshareValidators struct {
	NewValidators []Validator
}

// ParticipationProof represents the TypeParticipationProof mutation data.
type ParticipationProof struct {
	StartEpoch int
	EndEpoch   int
	Validators map[PublicKey]map[DutyType]map[PublicKey]int // map[validator]map[duty]map[operator]count
}

// AppendToCluster appends the mutation to the cluster, returning the new cluster state.
func AppendToCluster(sm SignedMutation, cluster Cluster) (Cluster, error) {
	m, ok := typeDef[sm.Mutation.Type]
	if !ok {
		return Cluster{}, fmt.Errorf("unknown mutation type: %s", sm.Mutation.Type)
	}

	resp, err := m.AppendFunc(sm, cluster.Clone())
	if err != nil {
		return Cluster{}, err
	}

	resp.Height++
	resp.Hashes[sm.Hash] = sm

	return resp, nil
}

type Approvals int

const (
	ApprovalsNone Approvals = iota
	ApprovalsQuorum
	ApprovalsAll
)

var typeDef = map[MutationType]struct {
	Approvals   Approvals
	DataType    any
	ParentTypes []MutationType
	AppendFunc  func(SignedMutation, Cluster) (Cluster, error)
}{
	TypeCreateCluster: {
		Approvals: ApprovalsNone,
		DataType:  CreateCluster{},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			cc, ok := m.Mutation.Data.(CreateCluster)
			if !ok {
				return Cluster{}, fmt.Errorf("invalid data type")
			}

			if cc.Name == "" || len(cc.Operators) == 0 { // cc.NumValidators == 0 || cc.WithdrawalAddress == ""
				return Cluster{}, fmt.Errorf("invalid create cluster mutation")
			}

			if c.Name != "" ||
				c.Height != 0 ||
				c.ApprovedMutations != 0 ||
				len(c.Operators) != 0 ||
				c.NumValidators != 0 ||
				c.WithdrawalAddress != "" ||
				len(c.Validators) != 0 ||
				len(c.ParticipationProof) != 0 {
				return Cluster{}, fmt.Errorf("cluster already exists")
			}

			var ops []Operator
			for _, op := range cc.Operators {
				ops = append(ops, Operator{
					PublicKey: op,
				})
			}

			return Cluster{
				ApprovedMutations: 1,
				Name:              cc.Name,
				Operators:         ops,
				NumValidators:     cc.NumValidators,
				WithdrawalAddress: cc.WithdrawalAddress,
			}, nil
		},
	},
	TypeOperatorENR: {
		Approvals:   ApprovalsNone,
		DataType:    OperatorENR{},
		ParentTypes: []MutationType{TypeCreateCluster, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			for i := 0; i < len(c.Operators); i++ {
				if c.Operators[i].PublicKey != m.Source {
					continue
				}

				if c.Operators[i].ENR != "" {
					return Cluster{}, fmt.Errorf("operator already has enr")
				}

				c.Operators[i].ENR = m.Mutation.Data.(OperatorENR).ENR

				return c, nil
			}

			return Cluster{}, fmt.Errorf("operator not found")
		},
	},
	TypeGenerateValidators: {
		Approvals:   ApprovalsAll,
		DataType:    GenerateValidators{},
		ParentTypes: []MutationType{TypeOperatorAck, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			missing := c.NumValidators - len(c.Validators)
			if missing <= 0 {
				return Cluster{}, fmt.Errorf("cluster already has all validators")
			}

			gv := m.Mutation.Data.(GenerateValidators)
			if len(gv.Validators) > missing {
				return Cluster{}, fmt.Errorf("too many validators")
			}

			for _, v := range gv.Validators {
				if len(v.PublicShares) != len(c.Operators) {
					return Cluster{}, fmt.Errorf("invalid validator")
				}
			}

			c.Validators = append(c.Validators, gv.Validators...)

			c.ApprovedMutations++

			return c, nil
		},
	},
	TypeAddValidators: {
		Approvals:   ApprovalsQuorum,
		DataType:    AddValidators{},
		ParentTypes: []MutationType{TypeOperatorAck, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			c.NumValidators += m.Mutation.Data.(AddValidators).NumValidators

			c.ApprovedMutations++

			return c, nil
		},
	},
	TypeOperatorAck: {
		Approvals:   ApprovalsNone,
		DataType:    OperatorAck{},
		ParentTypes: []MutationType{TypeAddValidators, TypeGenerateValidators, TypeReshareValidators},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {

			return c, nil
		},
	},
	TypeChangeOperators: {
		Approvals:   ApprovalsQuorum,
		DataType:    ChangeOperators{},
		ParentTypes: []MutationType{TypeOperatorAck, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			co := m.Mutation.Data.(ChangeOperators)
			if len(co.NewOperators) != len(c.Operators) {
				return Cluster{}, fmt.Errorf("invalid change operators")
			}

			var ops []Operator
			for _, op := range co.NewOperators {
				ops = append(ops, Operator{
					PublicKey: op,
				})
			}

			c.Operators = ops

			c.ApprovedMutations++

			return c, nil
		},
	},
	TypeReshareValidators: {
		Approvals:   ApprovalsAll,
		DataType:    ReshareValidators{},
		ParentTypes: []MutationType{TypeOperatorAck, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			for _, operator := range c.Operators {
				if operator.ENR == "" {
					return Cluster{}, fmt.Errorf("operator has no enr")
				}
			}
			rv := m.Mutation.Data.(ReshareValidators)
			if len(rv.NewValidators) != len(c.Validators) {
				return Cluster{}, fmt.Errorf("invalid reshare")
			}
			for i := 0; i < len(c.Validators); i++ {
				if len(rv.NewValidators[i].PublicShares) != len(c.Operators) {
					return Cluster{}, fmt.Errorf("invalid validator")
				}
				if rv.NewValidators[i].PublicKey != c.Validators[i].PublicKey {
					return Cluster{}, fmt.Errorf("invalid validator")
				}
			}
			c.Validators = rv.NewValidators

			c.ApprovedMutations++

			return c, nil
		},
	},
	TypeParticipationProof: {
		Approvals:   ApprovalsNone,
		DataType:    ParticipationProof{},
		ParentTypes: []MutationType{TypeParticipationProof, TypeOperatorAck, TypeOperatorENR},
		AppendFunc: func(m SignedMutation, c Cluster) (Cluster, error) {
			pp := m.Mutation.Data.(ParticipationProof)
			for _, prev := range c.ParticipationProof {
				if pp.StartEpoch >= prev.StartEpoch && pp.StartEpoch <= prev.EndEpoch {
					return Cluster{}, fmt.Errorf("overlapping participation proof")
				}
				if pp.EndEpoch >= prev.StartEpoch && pp.EndEpoch <= prev.EndEpoch {
					return Cluster{}, fmt.Errorf("overlapping participation proof")
				}
				for pval, counts := range pp.Validators {
					var found bool
					for _, cval := range c.Validators {
						if pval == cval.PublicKey {
							for _, operator := range c.Operators {
								for _, ops := range counts {
									_, ok := ops[operator.PublicKey]
									if !ok {
										return Cluster{}, fmt.Errorf("missing operator")
									}
								}
							}
							found = true
							break
						}
					}
					if !found {
						return Cluster{}, fmt.Errorf("unknown validator")
					}
				}
			}
			c.ParticipationProof = append(c.ParticipationProof, pp)

			return c.Clone(), nil
		},
	},
}
