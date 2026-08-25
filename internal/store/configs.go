package store

import (
	"fmt"
	"sort"

	bolt "go.etcd.io/bbolt"
	"internalauth/internal/domain"
)

func (s *Store) CreateRedisConfig(item domain.RedisConfig) error {
	if err := domain.ValidateRedisConfig(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		return putUnique(bucket, item.ID, item)
	})
}

func (s *Store) UpdateRedisConfig(item domain.RedisConfig) error {
	if err := domain.ValidateRedisConfig(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		return putReplace(bucket, item.ID, item)
	})
}

func (s *Store) GetRedisConfig(id string) (domain.RedisConfig, error) {
	var out domain.RedisConfig
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		return getOne(bucket, id, &out)
	})
	return out, err
}

func (s *Store) ListRedisConfigs() ([]domain.RedisConfig, error) {
	var out []domain.RedisConfig
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		out, err = listAll[domain.RedisConfig](bucket)
		return err
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, err
}

func (s *Store) DeleteRedisConfig(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		policies, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		err = policies.ForEach(func(_, value []byte) error {
			var p domain.RoutePolicy
			if err := decode(value, &p); err != nil {
				return err
			}
			if p.RedisConfigID == id {
				return fmt.Errorf("%w: redis config is used by policy %s", domain.ErrConflict, p.ID)
			}
			return nil
		})
		if err != nil {
			return err
		}
		bucket, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		return deleteOne(bucket, id)
	})
}

func (s *Store) CreatePolicy(item domain.RoutePolicy) error {
	if err := domain.ValidateRoutePolicy(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		configs, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		if configs.Get([]byte(item.RedisConfigID)) == nil {
			return fmt.Errorf("%w: redis config %s", domain.ErrNotFound, item.RedisConfigID)
		}
		bucket, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		return putUnique(bucket, item.ID, item)
	})
}

func (s *Store) UpdatePolicy(item domain.RoutePolicy) error {
	if err := domain.ValidateRoutePolicy(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		configs, err := requireBucket(tx, bucketRedis)
		if err != nil {
			return err
		}
		if configs.Get([]byte(item.RedisConfigID)) == nil {
			return fmt.Errorf("%w: redis config %s", domain.ErrNotFound, item.RedisConfigID)
		}
		bucket, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		return putReplace(bucket, item.ID, item)
	})
}

func (s *Store) GetPolicy(id string) (domain.RoutePolicy, error) {
	var out domain.RoutePolicy
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		return getOne(bucket, id, &out)
	})
	return out, err
}

func (s *Store) ListPolicies() ([]domain.RoutePolicy, error) {
	var out []domain.RoutePolicy
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		out, err = listAll[domain.RoutePolicy](bucket)
		return err
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, err
}

func (s *Store) DeletePolicy(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		return deleteOne(bucket, id)
	})
}
