package util

type Int64Set struct {
	members map[int64]struct{}
}

func EmptyInt64Set() *Int64Set {
	return &Int64Set{
		members: make(map[int64]struct{}),
	}
}

func Int64SetFromInt(x int64) *Int64Set {
	s := EmptyInt64Set()
	s.members[x] = struct{}{}
	return s
}

func Int64SetFromIntList(list []int64) *Int64Set {
	s := EmptyInt64Set()
	for _, x := range list {
		s.members[x] = struct{}{}
	}
	return s
}

func (s *Int64Set) Has(x int64) bool {
	if s == nil {
		return false
	}
	_, ok := s.members[x]
	return ok
}
