package clusterstate

import (
	"bytes"
	"fmt"
	"sort"
)

type State []SignedMutation

func (s State) Get(h Hash) (SignedMutation, int, error) {
	for i, m := range s {
		if m.Hash == h {
			return m, i, nil
		}
	}

	return SignedMutation{}, 0, fmt.Errorf("hash not found")
}

func (s State) Children(h Hash) ([]SignedMutation, error) {
	var resp []SignedMutation
	for _, m := range s {
		for _, p := range m.Mutation.ParentHashes {
			if p == h {
				resp = append(resp, m)
			}
		}
	}

	sort.Slice(resp, func(i, j int) bool {
		return bytes.Compare(resp[i].Hash[:], resp[j].Hash[:]) < 0
	})

	return resp, nil
}

func (s State) ApprovedBy(hash Hash) (map[PublicKey]bool, error) {
	children, err := s.Children(hash)
	if err != nil {
		return nil, err
	}

	resp := make(map[PublicKey]bool)
	for len(children) > 0 {
		child := children[0]
		resp[child.Source] = true

		more, err := s.Children(child.Hash)
		if err != nil {
			return nil, err
		}
		children = append(children, more...)
	}

	return resp, nil
}

func (s State) Leaves() []Hash {
	hasChildren := map[Hash]bool{}
	for _, m := range s {
		if _, ok := hasChildren[m.Hash]; !ok {
			hasChildren[m.Hash] = false
		}

		for _, p := range m.Mutation.ParentHashes {
			hasChildren[p] = true
		}
	}

	var resp []Hash
	for hash, ok := range hasChildren {
		if ok {
			continue
		}
		resp = append(resp, hash)
	}

	return resp
}

func (s State) Heights() (map[Hash]int, error) {
	buffer := []Hash{s[0].Hash}
	heights := map[Hash]int{
		s[0].Hash: 1,
	}

	for len(buffer) > 0 {
		h := buffer[0]
		buffer = buffer[1:]

		children, err := s.Children(h)
		if err != nil {
			return nil, err
		}
		if heights[h] == 0 {
			return nil, fmt.Errorf("bug: height not known")
		}

		childHeight := heights[h] + 1
		for _, child := range children {
			if heights[child.Hash] < childHeight {
				heights[child.Hash] = childHeight
			}

			buffer = append(buffer, child.Hash)
		}
	}

	return heights, nil
}

func (s State) Sequence(hash Hash) ([]SignedMutation, error) {
	heights, err := s.Heights()
	if err != nil {
		return nil, err
	}

	buffer := []Hash{hash}
	var resp []SignedMutation
	upstream := make(map[Hash]bool)
	for len(buffer) > 0 {
		h := buffer[0]
		buffer = buffer[1:]

		node, _, err := s.Get(h)
		if err != nil {
			return nil, err
		}

		resp = append(resp, node)

		for _, parent := range node.Mutation.ParentHashes {
			if upstream[parent] {
				continue
			}
			upstream[parent] = true
			buffer = append(buffer, parent)
		}
	}

	sort.Slice(resp, func(i, j int) bool {
		hi := heights[resp[i].Hash]
		hj := heights[resp[j].Hash]
		if hi != hj {
			return hi < hj
		}

		return bytes.Compare(resp[i].Hash[:], resp[j].Hash[:]) < 0
	})

	return resp, nil
}
