package buntdbx

import (
	"bytes"
	"encoding/gob"
	"strings"
	"time"

	"github.com/Oncelane/laneEtcd/src/common"
	"github.com/tidwall/buntdb"
)

type DB struct {
	db *buntdb.DB
}

func NewDB() *DB {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		return nil
	}
	return &DB{
		db: db,
	}
}

var _ common.DB = &DB{}

func (d *DB) Del(key string) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

func (d *DB) DelWithPrefix(prefix string) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		keys := make([]string, 0, 4)
		tx.AscendKeys(prefix+"*", func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
		for i := range keys {
			tx.Delete(keys[i])
		}
		return nil
	})
}

func (d *DB) Get(key string) (ret []byte, err error) {
	err = d.db.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(key)
		if err != nil {
			return err
		}
		ret = common.StringToBytes(v)
		return nil
	})
	if err == buntdb.ErrNotFound {
		err = common.ErrNotFound
	}
	return
}

func (d *DB) GetWithPrefix(prefix string) (ret [][]byte, err error) {
	ret = make([][]byte, 0, 1)
	err = d.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(prefix+"*", func(key, value string) bool {
			ret = append(ret, common.StringToBytes(value))
			return true
		})

	})
	return
}

func (d *DB) GetEntry(key string) (ret common.Entry, err error) {
	err = d.db.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(key)
		if err != nil {
			return err
		}
		err = gob.NewDecoder(strings.NewReader(v)).Decode(&ret)
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (d *DB) GetEntryWithPrefix(prefix string) (ret []common.Entry, err error) {
	ret = make([]common.Entry, 0, 1)

	err = d.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(prefix+"*", func(key, value string) bool {

			// 将 value 转换为 []byte
			data := common.StringToBytes(value)
			s := bytes.NewBuffer(data)

			dec := gob.NewDecoder(s)
			ret = append(ret, common.Entry{})
			err = dec.Decode(&ret[len(ret)-1])
			return err == nil
		})
	})
	return
}

func (d *DB) Put(key string, value []byte, DeadTime int64) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		if DeadTime != 0 {
			deadTime := time.UnixMilli(DeadTime)
			if deadTime.Before(time.Now()) {
				return nil
			}
			_, _, err := tx.Set(key, common.BytesToString(value), &buntdb.SetOptions{Expires: true, TTL: time.Until(deadTime)})
			if err != nil {
				return err
			}
		} else {
			_, _, err := tx.Set(key, common.BytesToString(value), nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) PutEntry(key string, entry common.Entry) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		b := &bytes.Buffer{}
		gob.NewEncoder(b).Encode(entry)
		if entry.DeadTime != 0 {
			deadTime := time.UnixMilli(entry.DeadTime)
			if deadTime.Before(time.Now()) {
				return nil
			}
			_, _, err := tx.Set(key, b.String(), &buntdb.SetOptions{Expires: true, TTL: time.Until(deadTime)})
			if err != nil {
				return err
			}
		} else {
			_, _, err := tx.Set(key, b.String(), nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) Keys(pageSize, pageIndex int) (ret []common.Pair, err error) {
	err = d.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			data := common.StringToBytes(value)
			s := bytes.NewBuffer(data)

			dec := gob.NewDecoder(s)
			ret = append(ret, common.Pair{Key: key})
			err = dec.Decode(&(ret[len(ret)-1].Entry))
			ret[len(ret)-1].Entry.Value = nil
			return err == nil
		})
	})
	return
}

func (d *DB) KVs(pageSize, pageIndex int) (ret []common.Pair, err error) {
	err = d.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			data := common.StringToBytes(value)
			s := bytes.NewBuffer(data)

			dec := gob.NewDecoder(s)
			ret = append(ret, common.Pair{Key: key})
			err = dec.Decode(&(ret[len(ret)-1].Entry))
			return err == nil
		})
	})
	return
}

func (d *DB) SnapshotData() ([]byte, error) {
	var b bytes.Buffer
	err := d.db.Save(&b)
	return b.Bytes(), err
}

func (d *DB) InstallSnapshotData(data []byte) error {
	d.db.Load(bytes.NewBuffer(data))
	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}
