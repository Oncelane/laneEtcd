package common

import (
	"errors"
	"unsafe"
)

// for vue3 frontend
type Respond struct {
	Code int `json:"code"`
	Data any `json:"data"`
}

type Entry struct {
	Value    []byte
	DeadTime int64
}

type Pair struct {
	Key   string
	Entry Entry
}

var (
	ErrNotFound = errors.New("not found")
)

type DB interface {
	Del(key string) error
	DelWithPrefix(prefix string) error
	Get(key string) ([]byte, error)
	Put(key string, value []byte, DeadTime int64) error
	GetWithPrefix(prefix string) ([][]byte, error)
	GetEntry(key string) (Entry, error)
	GetEntryWithPrefix(key string) ([]Entry, error)
	PutEntry(key string, entry Entry) error
	Keys(pageSize, pageIndex int) ([]Pair, error)
	KVs(pageSize, pageIndex int) ([]Pair, error)
	// Flush() error
	SnapshotData() ([]byte, error)
	InstallSnapshotData([]byte) error
	Close() error
}

func StringToBytes(s string) []byte {
	tmp1 := (*[2]uintptr)(unsafe.Pointer(&s))
	tmp2 := [3]uintptr{tmp1[0], tmp1[1], tmp1[1]}
	return *(*[]byte)(unsafe.Pointer(&tmp2))
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
