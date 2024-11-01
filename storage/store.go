package storage

import (
	"fmt"
	"path/filepath"
	"sync"
)

type Store struct {
	Memtable *Memtable
	Sstables []*SSTable
	lock     sync.RWMutex
	DestDir  string
}

func NewStore(maxSize int, desDir string) *Store {
	return &Store{
		Memtable: NewMemtable(maxSize),
		Sstables: []*SSTable{},
		DestDir:  desDir,
	}
}

func (s *Store) Put(key string, vector []float64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.Memtable.Put(key, vector, s.DestDir)
	if err != nil {
		return fmt.Errorf("could not Put data into memtable: %v", err)
	}

	if s.Memtable.Size() == 0 {
		err = s.loadNewSSTable()
		if err != nil {
			return fmt.Errorf("failed to load SSTable: %v", err)
		}
	}
	return nil
}

func (s *Store) Get(key string) ([]float64, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	vector, exists := s.Memtable.Get(key)
	if exists {
		return vector, true
	}

	for _, sstable := range s.Sstables {
		vector, exists, _ = sstable.Get(key)

		if exists {
			return vector, true
		}

	}

	return nil, false
}

func (s *Store) Flush() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Memtable.Size() > 0 {
		err := s.Memtable.flushToDisk(s.DestDir)
		if err != nil {
			return fmt.Errorf("could not Flush memtable data to Disk: %v", err)
		}
		s.Memtable.Clear()
		err = s.loadNewSSTable()
		if err != nil {
			return fmt.Errorf("could not load SSTable: %v", err)
		}
	}

	return nil
}

func (s *Store) loadNewSSTable() error {
	files, err := filepath.Glob(filepath.Join(s.DestDir, "*.sst"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil // No SSTables to load
	}

	// Find the newest SSTable file (assuming filenames are UUIDs)
	newestFile := files[len(files)-1]

	// Open the SSTable
	sstable, err := OpenSSTable(newestFile)
	if err != nil {
		return err
	}

	// Prepend to the list of SSTables
	s.Sstables = append([]*SSTable{sstable}, s.Sstables...)

	return nil
}
