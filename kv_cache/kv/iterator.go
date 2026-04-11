package kv

import "strings"

type Iterator struct {
	keys  []string
	index int
	db    *DB
}

func (db *DB) NewPrefixIterator(prefix string) *Iterator {
	var matchedKeys []string

	db.index.data.Range(func(k, v interface{}) bool {
		keyStr := k.(string)
		if strings.HasPrefix(keyStr, prefix) {
			matchedKeys = append(matchedKeys, keyStr)
		}
		return true
	})

	return &Iterator{
		keys:  matchedKeys,
		index: 0,
		db:    db,
	}
}

func (it *Iterator) Valid() bool {
	return it.index < len(it.keys)
}

func (it *Iterator) Next() {
	it.index++
}

func (it *Iterator) Value() ([]byte, error) {
	if !it.Valid() {
		return nil, nil
	}

	val, err := it.db.Get([]byte(it.keys[it.index]))
	return val, err
}
