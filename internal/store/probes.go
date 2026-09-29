package store

import (
	"context"
	"time"

	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

const (
	// probeListedFor keeps an offline probe in the list for a day.
	probeListedFor = 24 * time.Hour
	// probeOnlineWithin is how recent a heartbeat must be to count as online.
	probeOnlineWithin = 180
)

// UpsertProbe registers a probe or refreshes its heartbeat.
func (s *Store) UpsertProbe(ctx context.Context, name, region string, lastSeen int64) error {
	_, err := s.db.Exec(ctx, `INSERT INTO probes(name, region, last_seen) VALUES($1,$2,$3)
		ON CONFLICT(name) DO UPDATE SET region=excluded.region, last_seen=excluded.last_seen`, name, region, lastSeen)
	return err
}

// Probes lists probes seen within the last day.
func (s *Store) Probes(ctx context.Context) ([]types.Probe, error) {
	now := time.Now().Unix()
	rows, err := s.db.Query(ctx,
		`SELECT name, region, last_seen FROM probes WHERE last_seen >= $1 ORDER BY region, name`,
		now-int64(probeListedFor.Seconds()))
	if err != nil {
		return nil, err
	}
	return collect(rows, func(r pgx.Rows, p *types.Probe) error {
		if err := r.Scan(&p.Name, &p.Region, &p.LastSeen); err != nil {
			return err
		}
		p.Online = now-p.LastSeen < probeOnlineWithin
		return nil
	})
}
