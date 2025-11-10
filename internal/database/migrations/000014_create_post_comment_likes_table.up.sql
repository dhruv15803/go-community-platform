


CREATE TABLE IF NOT EXISTS post_comment_likes(
    liked_by_id INTEGER NOT NULL,
    liked_post_comment_id INTEGER NOT NULL,
    liked_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY(liked_by_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(liked_post_comment_id) REFERENCES post_comments(id) ON DELETE CASCADE,
    UNIQUE(liked_by_id,liked_post_comment_id)
);
