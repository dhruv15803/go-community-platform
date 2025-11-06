


CREATE TABLE IF NOT EXISTS post_comments(
    id SERIAL PRIMARY KEY,
    comment_content TEXT NOT NULL,
    comment_owner_id INTEGER NOT NULL,
    post_id INTEGER NOT NULL,
    parent_comment_id INTEGER NOT NULL,
    comment_created_at TIMESTAMP DEFAULT NOW(),
    comment_updated_at TIMESTAMP,
    FOREIGN KEY(comment_owner_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(parent_comment_id) REFERENCES  post_comments(id) ON DELETE CASCADE
);
