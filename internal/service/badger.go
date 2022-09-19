package service

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

var (
	db   *badger.DB
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const globalPrefix = "mixed-socks:global"

func New(path string) (err error) {
	var opt = badger.DefaultOptions(path)
	opt.Logger = logrus.StandardLogger()
	opt.WithCompression(options.ZSTD)
	opt.WithSyncWrites(true)
	opt.WithLoggingLevel(badger.ERROR)
	db, err = badger.Open(opt)
	if err != nil {
		logrus.Errorln(err)
	}
	return err
}

func Close() {
	var err = db.Close()
	if err != nil {
		logrus.Error(err)
	}
}

func Set(key string, data interface{}) error {
	value, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var fn = func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	}
	return db.Update(fn)
}

func Get(key string) (value []byte, err error) {
	var fn = func(tx *badger.Txn) error {
		var item *badger.Item
		item, err = tx.Get([]byte(key))
		if err != nil {
			return err
		}
		value, _ = item.ValueCopy(nil)
		return nil
	}

	err = db.View(fn)
	if err == badger.ErrKeyNotFound {
		return value, nil
	}
	return value, err
}

func Del(key string) error {
	var fn = func(tx *badger.Txn) error {
		return tx.Delete([]byte(key))
	}
	return db.Update(fn)
}

func List(prefix string) [][]byte {
	var value = make([][]byte, 0)
	_ = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			_value, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			value = append(value, _value)
		}
		return nil
	})
	return value
}
