package store

import (
	"context"

	"bangumi-status/internal/types"

	"github.com/jackc/pgx/v5"
)

// Reactions are live for 24h: every click refreshes its row's window.

// ReactionCounts aggregates active reactions per emoji. It is not
// user-specific, so callers may cache it.
func (s *Store) ReactionCounts(ctx context.Context) ([]types.ReactionCount, error) {
	rows, err := s.db.Query(ctx, `
SELECT emoji_id, SUM(count)::int FROM reactions
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY emoji_id ORDER BY emoji_id`)
	if err != nil {
		return nil, err
	}
	return collect(rows, func(r pgx.Rows, c *types.ReactionCount) error { return r.Scan(&c.EmojiID, &c.Count) })
}

// UserReactions returns the emoji ids userID has active reactions on.
func (s *Store) UserReactions(ctx context.Context, userID string) (map[int]bool, error) {
	out := map[int]bool{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.db.Query(ctx,
		`SELECT emoji_id::int FROM reactions WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`, userID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	for _, id := range ids {
		out[id] = true
	}
	return out, err
}

// AddReaction counts one more click of emojiID by userID.
func (s *Store) AddReaction(ctx context.Context, emojiID int, userID, ip string) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO reactions (emoji_id, user_id, ip, count) VALUES ($1, $2, $3, 1)
ON CONFLICT (emoji_id, user_id)
DO UPDATE SET count = reactions.count + 1, created_at = NOW(), ip = EXCLUDED.ip`, emojiID, userID, ip)
	return err
}

// PurgeExpiredReactions deletes rows past the 24h window. Reads already
// filter them out, so this is housekeeping only.
func (s *Store) PurgeExpiredReactions(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `DELETE FROM reactions WHERE created_at <= NOW() - INTERVAL '24 hours'`)
	return err
}
