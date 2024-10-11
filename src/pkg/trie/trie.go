package trie

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/Oncelane/laneEtcd/src/pkg/laneLog"

	"github.com/derekparker/trie"
)

type TrieX struct {
	tree *trie.Trie
}

func NewTrieX() *TrieX {
	return &TrieX{
		tree: trie.New(),
	}
}

// --------------------普通版本----------------------

func (t *TrieX) Get(key string) (any, bool) {
	n, ok := t.tree.Find(key)
	if !ok {
		return "", false
	}
	return n.Meta(), true
}

func (t *TrieX) GetWithPrefix(key string) []any {
	keys := t.tree.PrefixSearch(key)
	rt := make([]any, 0, len(keys))
	for _, key := range keys {
		v, _ := t.Get(key)
		rt = append(rt, v)
	}
	return rt
}

func (t *TrieX) Put(key string, value any) {
	t.tree.Add(key, value)
}

func (t *TrieX) Del(key string) {
	t.tree.Remove(key)
}

func (t *TrieX) Keys() []string {
	return t.tree.Keys()
}

// ------------------Entry版本------------------

// get额外承担惰性删除
func (t *TrieX) GetEntry(key string) (Entry, bool) {
	n, ok := t.tree.Find(key)
	if !ok {
		return Entry{}, false
	}
	rt, ok := n.Meta().(Entry)
	if !ok {
		laneLog.Logger.Fatalln("get value is not Entry")
		return Entry{}, false
	}
	// laneLog.Logger.Debugln("entry time:", rt.DeadTime, "timeNode:", time.Now().UnixMilli())
	if rt.DeadTime != 0 && rt.DeadTime < time.Now().UnixMilli() {
		// laneLog.Logger.Debugln("remove for dead")s
		t.tree.Remove(key)
		return Entry{}, false
	}
	return rt, true
}

func (t *TrieX) GetEntryWithPrefix(key string) []Entry {
	keys := t.tree.PrefixSearch(key)
	// 防止多次扩容
	rt := make([]Entry, 0, len(keys))
	for _, key := range keys {
		v, ok := t.GetEntry(key)
		if ok {
			rt = append(rt, v)
		}
	}
	return rt
}

func (t *TrieX) PutEntry(key string, value Entry) {
	t.tree.Add(key, value)
}

func (t *TrieX) Marshal() ([]byte, error) {

	w := new(bytes.Buffer)
	e := gob.NewEncoder(w)
	keys := t.tree.Keys()
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		value, _ := t.Get(key)
		values = append(values, value)
	}
	err := e.Encode(keys)
	if err != nil {
		return nil, err
	}
	err = e.Encode(values)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (t *TrieX) MarshalEncoder(e *gob.Encoder) error {
	keys := t.tree.Keys()
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		value, _ := t.Get(key)
		values = append(values, value)
	}
	err := e.Encode(keys)
	if err != nil {
		return err
	}
	err = e.Encode(values)
	if err != nil {
		return err
	}
	return nil
}

func (t *TrieX) UmMarshal(data []byte) error {
	t.tree = trie.New()
	w := bytes.NewBuffer(data)
	d := gob.NewDecoder(w)
	keys := make([]string, 0)
	values := make([]any, 0)
	err := d.Decode(&keys)
	if err != nil {
		return err
	}
	err = d.Decode(&values)
	if err != nil {
		return err
	}
	for i := range keys {
		t.Put(keys[i], values[i])
	}
	return nil
}
