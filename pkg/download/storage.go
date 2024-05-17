package download

import (
	"encoding/json"
	"go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

type Storage interface {
	Setup(buckets []string) error
	Put(bucket string, key string, v any) error
	Get(bucket string, key string, v any) (bool, error)
	List(bucket string, v any) error
	Pop(bucket string, key string, v any) error
	Delete(bucket string, key string) error

	Close() error
	Clear() error
}

func changeValue(p any, v any) {
	if v == nil {
		return
	}
	rp := reflect.ValueOf(p)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		if rv.Len() == 0 {
			return
		}
		// get underlying type
		tp := reflect.TypeOf(p).Elem().Elem()
		for i := 0; i < rv.Len(); i++ {
			// convert to underlying type
			vv := rv.Index(i).Elem().Convert(tp)
			rp.Elem().Set(reflect.Append(rp.Elem(), vv))
		}
	} else if rv.Kind() == reflect.Ptr {
		rp.Elem().Set(rv.Elem())
	} else {
		rp.Elem().Set(rv)
	}
}

var memData = make(map[string]map[string]any)

type MemStorage struct {
	lock *sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		lock: &sync.RWMutex{},
	}
}

func (n *MemStorage) Setup(buckets []string) error {
	for _, bucket := range buckets {
		if _, ok := memData[bucket]; !ok {
			memData[bucket] = make(map[string]any)
		}
	}
	return nil
}

func (n *MemStorage) Put(bucket string, key string, v any) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if bucketData, ok := memData[bucket]; ok {
		bucketData[key] = v
	}
	return nil
}

func (n *MemStorage) Get(bucket string, key string, v any) (bool, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	if dv, ok := memData[bucket][key]; ok {
		changeValue(v, dv)
		return true, nil
	}
	return false, nil
}

func (n *MemStorage) List(bucket string, v any) error {
	n.lock.RLock()
	defer n.lock.RUnlock()
	data := memData[bucket]
	list := make([]any, 0)
	for _, v := range data {
		list = append(list, v)
	}
	changeValue(v, list)
	return nil
}

func (n *MemStorage) Pop(bucket string, key string, v any) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	data := memData[bucket]
	changeValue(v, data[key])
	delete(data, key)
	return nil
}

func (n *MemStorage) Delete(bucket string, key string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	delete(memData[bucket], key)
	return nil
}

func (n *MemStorage) Close() error {
	return nil
}

func (n *MemStorage) Clear() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	memData = make(map[string]map[string]any)
	return nil
}

const (
	dbFile = "gopeed.db"
)

type BoltStorage struct {
	db   *bbolt.DB
	path string
}

func NewBoltStorage(dir string) *BoltStorage {
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	path := filepath.Join(dir, dbFile)
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		panic(err)
	}
	return &BoltStorage{
		db:   db,
		path: path,
	}
}

func (b *BoltStorage) Setup(buckets []string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BoltStorage) Put(bucket string, key string, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Put([]byte(key), buf)
	})
}

func (b *BoltStorage) Get(bucket string, key string, v any) (bool, error) {
	var data []byte
	err := b.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data = b.Get([]byte(key))
		return nil
	})
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, err
	}
	return true, nil
}

func (b *BoltStorage) List(bucket string, v any) error {
	list := make([]any, 0)
	tv := reflect.TypeOf(v).Elem().Elem()
	if err := b.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if err := b.ForEach(func(k, v []byte) error {
			data := reflect.New(tv.Elem()).Interface()
			if err := json.Unmarshal(v, &data); err != nil {
				return err
			}
			list = append(list, data)
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	changeValue(v, list)
	return nil
}

func (b *BoltStorage) Pop(bucket string, key string, v any) error {
	var data []byte
	err := b.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		kb := []byte(key)
		data = b.Get(kb)
		return b.Delete(kb)
	})
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func (b *BoltStorage) Delete(bucket string, key string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
}

func (b *BoltStorage) Close() error {
	return b.db.Close()
}

func (b *BoltStorage) Clear() error {
	if err := b.Close(); err != nil {
		return err
	}
	if err := os.Remove(b.path); err != nil {
		return err
	}
	return nil
}
