package storage

import (
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"math"
	"os"
	"path/filepath"
	"sync"
)

type Memtable struct {
	Data    *SkipList
	lock    sync.RWMutex
	maxSize int
	size    int
}

func NewMemtable(maxSize int) *Memtable {
	return &Memtable{
		Data:    NewSkipList(),
		maxSize: maxSize,
		size:    0,
	}
}

func (m *Memtable) Put(key string, vector []float64, destPath string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	var value []byte
	var err error

	if vector != nil {
		value, err = SerializeVector(vector)
		if err != nil {
			return fmt.Errorf("error when serializing data: %v", err)
		}
	}

	fmt.Printf("vectoer is serialized for memtable: %v", value)
	_, exists := m.Data.Search(key)
	if !exists {
		m.size++
	}

	m.Data.Insert(key, value)

	if m.size >= m.maxSize {
		err := m.flushToDisk(destPath)
		if err != nil {
			return fmt.Errorf("could not flush memtable to disk: %v", err)
		}
		m.Clear()
	}

	return nil
}

func SerializeVector(vector []float64) ([]byte, error) {
	buf := make([]byte, 8*len(vector))
	for i, v := range vector {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf, nil
}

func (m *Memtable) Get(key string) ([]float64, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	value, exists := m.Data.Search(key)
	if !exists {
		return nil, false
	}

	vector, err := DeserializeVector(value)
	if err != nil {
		panic(fmt.Errorf("could not deserialize vector: %v", err))
	}

	return vector, exists
}

func DeserializeVector(data []byte) ([]float64, error) {
	vector := make([]float64, len(data)/8)
	for i := range vector {
		bits := binary.LittleEndian.Uint64(data[i*8:])
		vector[i] = math.Float64frombits(bits)
	}

	return vector, nil
}

func (m *Memtable) Size() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.size
}

func (m *Memtable) flushToDisk(destPath string) error {
	err := os.MkdirAll(filepath.Dir(destPath), 0777)
	if err != nil {
		return fmt.Errorf("could not make path for %s: %v", destPath, err)
	}
	filename := filepath.Join(destPath, uuid.New().String()+".sst")
	tempFilename := filename + ".tmp"

	file, err := os.Create(tempFilename)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", tempFilename, err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	current := m.Data.head.next[0]
	for current != nil {
		err := writeRecord(file, current.key, current.value)
		if err != nil {
			return fmt.Errorf("could not write record: %v", err)
		}
		current = current.next[0]
	}

	err = os.Rename(tempFilename, filename)
	if err != nil {
		return fmt.Errorf("could not rename temp file to sst: %v", err)
	}

	return nil
}

func writeRecord(file *os.File, key string, value []byte) error {
	err := binary.Write(file, binary.LittleEndian, int32(len(key)))
	if err != nil {
		return err
	}

	_, err = file.Write([]byte(key))
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, int32(len(value)))
	if err != nil {
		return err
	}

	_, err = file.Write(value)
	if err != nil {
		return err
	}

	return nil
}

func (m *Memtable) Clear() {
	m.Data = NewSkipList()
	m.size = 0
}
