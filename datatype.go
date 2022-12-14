package main

type StdOut string
type StdErr string

/**
HashSet
*/
type HashSetMap map[string]struct{}

type HashSet struct {
	set HashSetMap
}

func NewHashSet() *HashSet {
	hashset := HashSet{
		set: make(HashSetMap),
	}

	return &hashset
}

func (hs *HashSet) Contains(key string) bool {
	_, contains := hs.set[key]
	return contains
}

func (hs *HashSet) Add(key string) {
	hs.set[key] = struct{}{}
}

func (hs *HashSet) Remove(key string) {
	delete(hs.set, key)
}

func (hs *HashSet) GetKeys() (keys []string) {
	keys = make([]string, len(hs.set))
	for k, _ := range hs.set {
		keys = append(keys, k)
	}
	return keys
}
