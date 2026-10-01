-- name: FindOpenVote :one
SELECT id FROM admin_votes
WHERE action_type = $1
  AND target_user_id = $2
  AND status = 'Open'
  AND deleted_at IS NULL;

-- name: InsertAdminVote :one
INSERT INTO admin_votes (
    action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, created_at;

-- name: InsertVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES ($1, $2, $3, $4);

-- name: FindUserRoleForVote :one
SELECT role FROM identity_users
WHERE id = $1 AND deleted_at IS NULL;

-- name: ResolveVoteImmediately :execrows
-- Used only when required_votes = 1 (not our case today, but the schema allows it).
UPDATE admin_votes
SET status = 'Resolved', resolved_at = $2, updated_at = now()
WHERE id = $1 AND status = 'Open' AND admin_votes.xmin::text = $3;