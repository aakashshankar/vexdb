package storage

import "math/rand"

const (
	maxLevel = 32
	p        = 0.25
)

type SkipNode struct {
	key   string
	value []byte
	next  []*SkipNode
}

type SkipList struct {
	head   *SkipNode
	level  int
	length int
}

func NewSkipList() *SkipList {
	return &SkipList{
		head: &SkipNode{
			next: make([]*SkipNode, maxLevel),
		},
		level: 0,
	}
}

func (s *SkipList) randomLevel() int {
	level := 0
	// Keep going while random number < p and we haven't reached maxLevel
	for rand.Float64() < p && level < maxLevel-1 {
		level++
	}
	return level
}

func (s *SkipList) Search(key string) ([]byte, bool) {
	current := s.head

	for i := s.level; i >= 0; i-- {
		for current.next[i] != nil && current.key < key {
			current = current.next[i]
		}
	}

	current = current.next[0]
	if current != nil && current.key == key {
		return current.value, true
	}

	return nil, false
}

func (s *SkipList) Iterator() func() (string, []byte, bool) {
	curr := s.head.next[0]
	return func() (string, []byte, bool) {
		if curr == nil {
			return "", nil, false
		}
		key := curr.key
		vector := curr.value
		curr = curr.next[0]
		return key, vector, true
	}
}

func (s *SkipList) Insert(key string, value []byte) {
	update := make([]*SkipNode, maxLevel)
	current := s.head

	// Start from the highest level and work down
	for i := s.level; i >= 0; i-- {
		// Move forward while next node's key is less than insert key
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	// Move to the next node at base level
	current = current.next[0]

	// If key already exists, update the value
	if current != nil && current.key == key {
		current.value = value
		return
	}

	// Generate a random level for the new node
	newLevel := s.randomLevel()

	// If the new level is higher than the current list level,
	// update the update array with the head node for those levels
	if newLevel > s.level {
		for i := s.level + 1; i <= newLevel; i++ {
			update[i] = s.head
		}
		s.level = newLevel
	}

	// Create new node
	newNode := &SkipNode{
		key:   key,
		value: value,
		next:  make([]*SkipNode, newLevel+1),
	}

	// Insert the node at all levels
	for i := 0; i <= newLevel; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}

	s.length++

}
