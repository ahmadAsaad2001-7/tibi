-- name: GetVoteForUpdate :one
SELECT
    id, action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at,
    xmin::text AS xmin
FROM admin_votes
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetVoteParticipants :many
SELECT id, admin_vote_id, admin_user_id, vote, voted_at
FROM admin_vote_participants
WHERE admin_vote_id = $1
ORDER BY voted_at;

-- name: UpdateVoteTally :execrows
UPDATE admin_votes
SET votes_for     = $2,
    votes_against = $3,
    status        = $4,
    resolved_at   = $5,
    updated_at    = now()
WHERE id = $1 AND xmin::text = $6;

-- name: InsertVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES ($1, $2, $3, $4);