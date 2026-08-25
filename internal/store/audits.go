package store

import (
	"fmt"
	"sort"

	bolt "go.etcd.io/bbolt"
	"internalauth/internal/domain"
)

func (s *Store) AppendAudit(item domain.AuditEvent) error {
	if err := domain.ValidateAuditEvent(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketAudits)
		if err != nil {
			return err
		}
		return putUnique(bucket, item.ID, item)
	})
}

func (s *Store) GetAudit(id string) (domain.AuditEvent, error) {
	var out domain.AuditEvent
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketAudits)
		if err != nil {
			return err
		}
		return getOne(bucket, id, &out)
	})
	return out, err
}

func (s *Store) ListAudits(filter domain.AuditFilter) ([]domain.AuditEvent, error) {
	var out []domain.AuditEvent
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketAudits)
		if err != nil {
			return err
		}
		out, err = listAll[domain.AuditEvent](bucket)
		return err
	})
	if err != nil {
		return nil, err
	}
	return domain.FilterAudits(out, filter), nil
}

func (s *Store) SaveDeployment(item domain.Deployment) error {
	if err := domain.ValidateDeployment(item); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		policies, err := requireBucket(tx, bucketPolicies)
		if err != nil {
			return err
		}
		if policies.Get([]byte(item.RouteID)) == nil {
			return fmt.Errorf("%w: route policy %s", domain.ErrNotFound, item.RouteID)
		}
		bucket, err := requireBucket(tx, bucketDeployments)
		if err != nil {
			return err
		}
		return putUnique(bucket, item.ID, item)
	})
}

func (s *Store) GetDeployment(id string) (domain.Deployment, error) {
	var out domain.Deployment
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketDeployments)
		if err != nil {
			return err
		}
		return getOne(bucket, id, &out)
	})
	return out, err
}

func (s *Store) ListDeployments(routeID string) ([]domain.Deployment, error) {
	var all []domain.Deployment
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket, err := requireBucket(tx, bucketDeployments)
		if err != nil {
			return err
		}
		all, err = listAll[domain.Deployment](bucket)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Deployment, 0)
	for _, item := range all {
		if routeID == "" || item.RouteID == routeID {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RouteID == out[j].RouteID {
			return out[i].Revision > out[j].Revision
		}
		return out[i].RouteID < out[j].RouteID
	})
	return out, nil
}

func (s *Store) Dashboard() (domain.Dashboard, error) {
	var out domain.Dashboard
	err := s.db.View(func(tx *bolt.Tx) error {
		for name, target := range map[string]*int{"route_policies": &out.Policies, "redis_configs": &out.RedisConfigs} {
			bucket := tx.Bucket([]byte(name))
			if bucket == nil {
				return fmt.Errorf("bucket %s missing", name)
			}
			*target = bucket.Stats().KeyN
		}
		tokens, err := requireBucket(tx, bucketTokens)
		if err != nil {
			return err
		}
		if err := tokens.ForEach(func(_, value []byte) error {
			var item domain.TokenRecord
			if err := decode(value, &item); err != nil {
				return err
			}
			if !item.Revoked {
				out.ActiveTokens++
			}
			return nil
		}); err != nil {
			return err
		}
		audits, err := requireBucket(tx, bucketAudits)
		if err != nil {
			return err
		}
		return audits.ForEach(func(_, value []byte) error {
			var item domain.AuditEvent
			if err := decode(value, &item); err != nil {
				return err
			}
			switch item.Outcome {
			case "allowed":
				out.Allowed++
			case "denied":
				out.Denied++
			case "unavailable":
				out.Unavailable++
			}
			return nil
		})
	})
	return out, err
}
