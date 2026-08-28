package store

import (
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
	"internalauth/internal/domain"
)

func (s *Store) CreateToken(item domain.TokenRecord) error {
	if err := domain.ValidateTokenRecord(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		return putUnique(bucket, item.ID, item)
	})
}

func (s *Store) UpdateToken(item domain.TokenRecord) error {
	if err := domain.ValidateTokenRecord(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		return putReplace(bucket, item.ID, item)
	})
}

func (s *Store) GetToken(id string) (domain.TokenRecord, error) {
	var out domain.TokenRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		return getOne(bucket, id, &out)
	})
	return out, err
}

func (s *Store) FindTokenByHash(hash string) (domain.TokenRecord, error) {
	var out domain.TokenRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		found := false
		err = bucket.ForEach(func(_, value []byte) error {
			var item domain.TokenRecord
			if err := decode(value, &item); err != nil {
				return err
			}
			if item.TokenHash == hash {
				out = item
				found = true
			}
			return nil
		})
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrNotFound
		}
		return nil
	})
	return out, err
}

func (s *Store) ListTokens() ([]domain.TokenRecord, error) {
	var out []domain.TokenRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		out, err = listAll[domain.TokenRecord](bucket)
		return err
	})
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, err
}

func (s *Store) RevokeToken(id string, now time.Time) (domain.TokenRecord, error) {
	var out domain.TokenRecord
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		if err := getOne(bucket, id, &out); err != nil {
			return err
		}
		if out.Revoked {
			return nil
		}
		out.Revoked = true
		out.UpdatedAt = now
		return putReplace(bucket, id, out)
	})
	return out, err
}

func (s *Store) DeleteExpiredTokens(now time.Time) (int, error) {
	count := 0
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		keys := make([][]byte, 0)
		if err := bucket.ForEach(func(key, value []byte) error {
			var item domain.TokenRecord
			if err := decode(value, &item); err != nil {
				return err
			}
			if !item.ExpiresAt.After(now) {
				keys = append(keys, append([]byte(nil), key...))
			}
			return nil
		}); err != nil {
			return err
		}
		for _, key := range keys {
			if err := bucket.Delete(key); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}
