package util

type IntSet struct {
	members map[int]struct{}
}

func EmptyIntSet() *IntSet {
	return &IntSet{
		members: make(map[int]struct{}),
	}
}

func IntSetFromInt(x int) *IntSet {
	s := EmptyIntSet()
	s.members[x] = struct{}{}
	return s
}

func IntSetFromIntList(list []int) *IntSet {
	s := EmptyIntSet()
	for _, x := range list {
		s.members[x] = struct{}{}
	}
	return s
}

func (s *IntSet) Has(x int) bool {
	if s == nil {
		return false
	}
	_, ok := s.members[x]
	return ok
}
