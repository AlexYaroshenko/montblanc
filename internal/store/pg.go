package store

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type PgStore struct {
    pool *pgxpool.Pool
}

func OpenPostgres(ctx context.Context, url string) (*PgStore, error) {
    pool, err := pgxpool.New(ctx, url)
    if err != nil { return nil, err }
    s := &PgStore{pool: pool}
    if err := s.init(ctx); err != nil {
        pool.Close()
        return nil, err
    }
    return s, nil
}

func (s *PgStore) init(ctx context.Context) error {
    stmts := []string{
        `create table if not exists subscribers (
            chat_id text primary key,
            language text not null default 'en',
            plan text not null default 'free',
            is_active boolean not null default true,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
        `create table if not exists subscriptions (
            id text primary key,
            chat_id text not null references subscribers(chat_id) on delete cascade,
            refuge text not null,
            date_from text,
            date_to text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
    }
    for _, q := range stmts {
        if _, err := s.pool.Exec(ctx, q); err != nil { return err }
    }
    return nil
}

func (s *PgStore) Close() error { s.pool.Close(); return nil }

func (s *PgStore) UpsertSubscriber(sub Subscriber) error {
    now := time.Now()
    if sub.CreatedAt.IsZero() { sub.CreatedAt = now }
    sub.LastUpdatedAt = now
    if sub.Plan == "" { sub.Plan = "free" }
    _, err := s.pool.Exec(context.Background(),
        `insert into subscribers (chat_id, language, plan, is_active, created_at, updated_at)
         values ($1,$2,$3,$4,$5,$6)
         on conflict (chat_id) do update set language=excluded.language, plan=excluded.plan, is_active=excluded.is_active, updated_at=excluded.updated_at`,
        sub.ChatID, sub.Language, sub.Plan, sub.IsActive, sub.CreatedAt, sub.LastUpdatedAt,
    )
    return err
}

func (s *PgStore) GetSubscriber(chatID string) (Subscriber, error) {
    var sub Subscriber
    err := s.pool.QueryRow(context.Background(),
        `select chat_id, language, plan, is_active, created_at, updated_at from subscribers where chat_id=$1`, chatID,
    ).Scan(&sub.ChatID, &sub.Language, &sub.Plan, &sub.IsActive, &sub.CreatedAt, &sub.LastUpdatedAt)
    if err != nil { return Subscriber{}, err }
    return sub, nil
}

func (s *PgStore) ListSubscribers() ([]Subscriber, error) {
    rows, err := s.pool.Query(context.Background(),
        `select chat_id, language, plan, is_active, created_at, updated_at from subscribers where is_active=true`)
    if err != nil { return nil, err }
    defer rows.Close()
    var subs []Subscriber
    for rows.Next() {
        var sub Subscriber
        if err := rows.Scan(&sub.ChatID, &sub.Language, &sub.Plan, &sub.IsActive, &sub.CreatedAt, &sub.LastUpdatedAt); err != nil { return nil, err }
        subs = append(subs, sub)
    }
    return subs, rows.Err()
}

func (s *PgStore) DeactivateSubscriber(chatID string) error {
    _, err := s.pool.Exec(context.Background(), `update subscribers set is_active=false, updated_at=now() where chat_id=$1`, chatID)
    return err
}

func (s *PgStore) AddQuery(q Query) (string, error) {
    if q.ID == "" { q.ID = q.ChatID + "-" + time.Now().Format("20060102150405.000000000") }
    _, err := s.pool.Exec(context.Background(),
        `insert into subscriptions (id, chat_id, refuge, date_from, date_to, created_at, updated_at)
         values ($1,$2,$3,$4,$5, now(), now())`,
        q.ID, q.ChatID, q.Refuge, q.DateFrom, q.DateTo,
    )
    if err != nil { return "", err }
    return q.ID, nil
}

func (s *PgStore) ListQueriesByChat(chatID string) ([]Query, error) {
    rows, err := s.pool.Query(context.Background(),
        `select id, chat_id, refuge, date_from, date_to, created_at, updated_at from subscriptions where chat_id=$1`, chatID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []Query
    for rows.Next() {
        var q Query
        if err := rows.Scan(&q.ID, &q.ChatID, &q.Refuge, &q.DateFrom, &q.DateTo, &q.CreatedAt, &q.LastUpdatedAt); err != nil { return nil, err }
        res = append(res, q)
    }
    return res, rows.Err()
}


