package v5

type ClusterState struct {
	Name       string
	Operators  []Operator
	Validators []Validator
}

type Operator struct {
	PublicKey PublicKey
	ENR       string
}
