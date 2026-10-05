-- name: InsertPost :one
INSERT INTO content_posts (
    doctor_profile_id, title, content, excerpt, type, cover_image_url, is_published
) VALUES (
             @doctor_profile_id, @title, @content, @excerpt, @type, @cover_image_url, false
         ) RETURNING id, created_at;

-- name: InsertPostAttachment :exec
INSERT INTO content_post_attachments (doctor_post_id, file_id)
VALUES (@doctor_post_id, @file_id);