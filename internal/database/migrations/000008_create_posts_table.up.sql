


CREATE TABLE IF NOT EXISTS posts(
    id SERIAL PRIMARY KEY,
    post_title TEXT NOT NULL,
    post_content TEXT NOT NULL,
    post_owner_id INTEGER NOT NULL,
    post_community_id INTEGER NOT NULL,
    post_created_at TIMESTAMP DEFAULT NOW(),
    post_updated_at TIMESTAMP,
    FOREIGN KEY(post_owner_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_community_id) REFERENCES communities(id) ON DELETE CASCADE
);
