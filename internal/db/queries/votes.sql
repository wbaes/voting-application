-- name: CastVote :one
INSERT INTO votes (photo_id, session_id, name, email, phone)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetVoteBySession :one
SELECT * FROM votes WHERE session_id = ?;

-- name: GetVoteCounts :many
SELECT photo_id, COUNT(*) AS vote_count
FROM votes
GROUP BY photo_id;

-- name: GetTotalVotes :one
SELECT COUNT(*) AS total FROM votes;

-- name: GetVotersWithContact :many
SELECT * FROM votes
WHERE name != '' OR email != '' OR phone != '';

-- name: GetVoteByID :one
SELECT * FROM votes WHERE id = ?;

-- name: ClearAllVotes :exec
DELETE FROM votes;

-- name: ClearAllDrawResults :exec
DELETE FROM draw_results;

-- name: RecordDrawResult :one
INSERT INTO draw_results (vote_id)
VALUES (?)
RETURNING *;

-- name: GetDrawResults :many
SELECT dr.id, dr.vote_id, dr.drawn_at,
       v.photo_id, v.name, v.email, v.phone
FROM draw_results dr
JOIN votes v ON v.id = dr.vote_id
ORDER BY dr.drawn_at DESC;
