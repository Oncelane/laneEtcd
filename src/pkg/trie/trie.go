package trie

import (
	"bytes"
	"encoding/gob"

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

func (t *TrieX) Get(key string) (string, bool) {
	n, ok := t.tree.Find(key)
	if !ok {
		return "", false
	}
	rt, ok := n.Meta().(string)
	if !ok {
		laneLog.Logger.Fatalln("get value is not string")
		return "", false
	}
	return rt, true
}

func (t *TrieX) GetWithPrefix(key string) []string {
	keys := t.tree.PrefixSearch(key)
	var rt []string
	for _, key := range keys {
		v, _ := t.Get(key)
		rt = append(rt, v)
	}
	return rt
}

func (t *TrieX) Put(key, value string) {
	t.tree.Add(key, value)
}

func (t *TrieX) Del(key string) {
	t.tree.Remove(key)
}

func (t *TrieX) Keys() []string {
	return t.tree.Keys()
}

func (t *TrieX) Marshal() ([]byte, error) {

	w := new(bytes.Buffer)
	e := gob.NewEncoder(w)
	keys := t.tree.Keys()
	values := make([]string, 0)
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

func (t *TrieX) UmMarshal(data []byte) error {
	t.tree = trie.New()
	w := bytes.NewBuffer(data)
	d := gob.NewDecoder(w)
	keys := make([]string, 0)
	values := make([]string, 0)
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
