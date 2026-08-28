package store

import (
	"encoding/json"
	"errors"
	"go.etcd.io/bbolt"
)

type Options struct{}
type Store struct{ db *bbolt.DB }

var bucketRedis = []byte("redis_configs")
var bucketPolicies = []byte("route_policies")
var bucketTokens = []byte("tokens")
var bucketAudits = []byte("audits")
var bucketDeployments = []byte("deployments")

func Open(path string, _ Options) (*Store, error) {
	db, e := bbolt.Open(path, 0600, nil)
	if e != nil {
		return nil, e
	}
	s := &Store{db: db}
	e = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketRedis, bucketPolicies, bucketTokens, bucketAudits, bucketDeployments} {
			if _, x := tx.CreateBucketIfNotExists(b); x != nil {
				return x
			}
		}
		return nil
	})
	if e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
func requireBucket(tx *bbolt.Tx, name []byte) (*bbolt.Bucket, error) {
	b := tx.Bucket(name)
	if b == nil {
		return nil, errors.New("bucket missing")
	}
	return b, nil
}
func encode(v any) ([]byte, error) { return json.Marshal(v) }
func decode(b []byte, v any) error { return json.Unmarshal(b, v) }
func putUnique(b *bbolt.Bucket, id string, v any) error {
	if b.Get([]byte(id)) != nil {
		return errors.New("already exists")
	}
	raw, e := encode(v)
	if e != nil {
		return e
	}
	return b.Put([]byte(id), raw)
}
func putReplace(b *bbolt.Bucket, id string, v any) error {
	raw, e := encode(v)
	if e != nil {
		return e
	}
	return b.Put([]byte(id), raw)
}
func getOne(b *bbolt.Bucket, id string, v any) error {
	raw := b.Get([]byte(id))
	if raw == nil {
		return errors.New("not found")
	}
	return decode(raw, v)
}

func deleteOne(b *bbolt.Bucket, id string) error { return b.Delete([]byte(id)) }
func listAll[T any](b *bbolt.Bucket) ([]T, error) {
	out := []T{}
	e := b.ForEach(func(_, raw []byte) error {
		var v T
		if e := decode(raw, &v); e != nil {
			return e
		}
		out = append(out, v)
		return nil
	})
	return out, e
}
