package trie_test

import (
	"fmt"
	"log"
	"strings"
	"testing"

	trieX "github.com/Oncelane/laneEtcd/src/pkg/trie"
	"github.com/derekparker/trie"
)

func TestTeri(test *testing.T) {
	t := trieX.NewTrieX()
	t.Put("a", "1")
	t.Put("ab", "2")
	t.Put("abb", "3")
	t.Put("b", "4")
	n, ok := t.Get("a")
	log.Println("v = ", n, "ok =", ok)
	n, ok = t.Get("c")
	log.Println("v = ", n, "ok =", ok)
	data, err := t.Marshal()
	if err != nil {
		test.Error(err)
	}
	err = t.UmMarshal(data)
	if err != nil {
		test.Error(err)
	}
	rt := t.GetWithPrefix("a")
	log.Println("v = ", rt)

}

func TestKeys(test *testing.T) {
	t := trie.New()
	t.Add("a", "1")
	t.Add("ab", "2")
	t.Add("abb", "3")
	t.Add("b", "4")
	var lookup func(*trie.Node, int)
	lookup = func(r *trie.Node, deepth int) {
		switch rune(r.Val()) {
		case rune(0):
			fmt.Printf("%s", strings.Repeat("\t", deepth))
		default:
			fmt.Printf("%s", strings.Repeat("\t", deepth+1))
			fmt.Printf("%c ", r.Val())
		}
		switch (r.Meta()).(type) {
		case nil:
			fmt.Println()
		case string:
			fmt.Printf("value %s\n", (r.Meta()).(string))
		}
		for _, v := range r.Children() {
			lookup(v, deepth+1)
		}
	}
	lookup(t.Root(), 0)
}
