-- name: FindOpenVote :one
SELECT id FROM admin_votes
WHERE action_type = @action_type
  AND target_user_id = @target_user_id
  AND status = 'Open'
  AND deleted_at IS NULL;

-- name: InsertAdminVote :one
INSERT INTO admin_votes (
    action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at
)
VALUES (
           @action_type, @target_user_id, @status, @required_votes,
           @votes_for, @votes_against, @expires_at, @resolved_at
       )
    RETURNING id, created_at;

-- name: InsertVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES (@admin_vote_id, @admin_user_id, @vote, @voted_at);

-- name: FindUserRoleForVote :one
SELECT role FROM identity_users
WHERE id = @id AND deleted_at IS NULL;