package trie

import (
	"bytes"
	"encoding/gob"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"
)

type Entry struct {
	Value    []byte
	DeadTime int64
}

func (e *Entry) Marshal() []byte {
	b := new(bytes.Buffer)
	en := gob.NewEncoder(b)
	err := en.Encode(e)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
	return b.Bytes()
}

func (e *Entry) Unmarshal(data []byte) {
	b := bytes.NewBuffer(data)
	de := gob.NewDecoder(b)
	err := de.Decode(e)
	if err != nil {
		laneLog.Logger.Fatalln(err)
	}
}
