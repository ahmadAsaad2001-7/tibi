-- name: GetSuspensionTarget :one
SELECT u.id AS user_id, u.role
FROM identity_users u
WHERE u.id = $1 AND u.deleted_at IS NULL;

-- name: HasActiveSuspension :one
SELECT EXISTS (
    SELECT 1 FROM admin_user_suspensions
    WHERE user_id = $1
      AND lifted_at IS NULL
      AND (to_ts IS NULL OR to_ts > now())
) AS has_active;

-- name: InsertSuspensionVote :one
INSERT INTO admin_votes (
    action_type, target_user_id, status,
    required_votes, votes_for, votes_against, expires_at, payload
) VALUES ('BanUser', $1, 'Open', $2, 1, 0, $3, $4)
RETURNING id, created_at;

-- name: InsertSuspensionVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES ($1, $2, 'For', $3);