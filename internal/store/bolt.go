package store

import (
    "encoding/json"
    "fmt"
    "os"
    "time"

    bolt "go.etcd.io/bbolt"
)

type BoltStore struct {
    db             *bolt.DB
    bktSubscribers []byte
    bktQueries     []byte
}

var (
	bucketSubscribers = []byte("subscribers")
	bucketQueries     = []byte("queries")
)

func OpenBolt(path string) (*BoltStore, error) {
    db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        return nil, err
    }
    prefix := os.Getenv("DB_TABLE_PREFIX")
    bktSubs := []byte(prefix + string(bucketSubscribers))
    bktQs := []byte(prefix + string(bucketQueries))
    err = db.Update(func(tx *bolt.Tx) error {
        if _, e := tx.CreateBucketIfNotExists(bktSubs); e != nil {
            return e
        }
        if _, e := tx.CreateBucketIfNotExists(bktQs); e != nil {
            return e
        }
        return nil
    })
    if err != nil {
        _ = db.Close()
        return nil, err
    }
    return &BoltStore{db: db, bktSubscribers: bktSubs, bktQueries: bktQs}, nil
}

func (s *BoltStore) Close() error { return s.db.Close() }

func (s *BoltStore) UpsertSubscriber(sub Subscriber) error {
	if sub.ChatID == "" {
		return fmt.Errorf("chat id required")
	}
	now := time.Now()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.LastUpdatedAt = now
	if sub.Plan == "" {
		sub.Plan = "free"
	}
	sub.IsActive = true
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}
    return s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket(s.bktSubscribers)
        return bucket.Put([]byte(sub.ChatID), b)
    })
}

func (s *BoltStore) GetSubscriber(chatID string) (Subscriber, error) {
	var sub Subscriber
    err := s.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(s.bktSubscribers)
        v := b.Get([]byte(chatID))
        if v == nil {
            return ErrNotFound
        }
        return json.Unmarshal(v, &sub)
    })
	return sub, err
}

func (s *BoltStore) ListSubscribers() ([]Subscriber, error) {
	var subs []Subscriber
    err := s.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(s.bktSubscribers)
        return b.ForEach(func(k, v []byte) error {
            var s Subscriber
            if err := json.Unmarshal(v, &s); err != nil {
                return err
            }
            subs = append(subs, s)
            return nil
        })
    })
	return subs, err
}

func (s *BoltStore) DeactivateSubscriber(chatID string) error {
	sub, err := s.GetSubscriber(chatID)
	if err != nil {
		return err
	}
	sub.IsActive = false
	return s.UpsertSubscriber(sub)
}

func (s *BoltStore) AddQuery(q Query) (string, error) {
	now := time.Now()
	q.CreatedAt = now
	q.LastUpdatedAt = now
	if q.ID == "" {
		q.ID = fmt.Sprintf("%s-%d", q.ChatID, now.UnixNano())
	}
	b, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
    err = s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket(s.bktQueries)
        return bucket.Put([]byte(q.ID), b)
    })
	if err != nil {
		return "", err
	}
	return q.ID, nil
}

func (s *BoltStore) ListQueriesByChat(chatID string) ([]Query, error) {
	var res []Query
    err := s.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(s.bktQueries)
        return b.ForEach(func(k, v []byte) error {
            var q Query
            if err := json.Unmarshal(v, &q); err != nil {
                return err
            }
            if q.ChatID == chatID {
                res = append(res, q)
            }
            return nil
        })
    })
	return res, err
}
