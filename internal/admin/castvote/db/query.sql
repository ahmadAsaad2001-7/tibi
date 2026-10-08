-- name: GetVoteForUpdate :one
SELECT
    id, action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at, payload,
    xmin::text AS xmin
FROM admin_votes
WHERE id = @id AND deleted_at IS NULL;

-- name: GetVoteParticipants :many
SELECT id, admin_vote_id, admin_user_id, vote, voted_at
FROM admin_vote_participants
WHERE admin_vote_id = @admin_vote_id
ORDER BY voted_at;

-- name: UpdateVoteTally :execrows
UPDATE admin_votes
SET votes_for     = @votes_for,
    votes_against = @votes_against,
    status        = @status,
    resolved_at   = @resolved_at,
    updated_at    = now()
WHERE id = @id AND xmin::text = @xmin;

-- name: InsertVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES (@admin_vote_id, @admin_user_id, @vote, @voted_at);