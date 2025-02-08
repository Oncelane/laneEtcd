package trie_test

import (
	"fmt"
	"testing"

	"github.com/tidwall/buntdb"
)

func TestBultDB(t *testing.T) {

	db, err := buntdb.Open("test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.Update(func(tx *buntdb.Tx) error {
		tx.Set("n:1", "one", nil)
		tx.Set("n:2", "two", nil)
		tx.Set("n:3", "three", nil)
		v, _ := tx.Get("n:1")
		fmt.Println("val", v)
		return nil
	})

}
