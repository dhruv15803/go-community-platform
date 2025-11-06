


CREATE TABLE IF NOT EXISTS user_communities(
    user_id INTEGER NOT NULL,
    community_id INTEGER NOT NULL,
    joined_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(community_id) REFERENCES communities(id) ON DELETE CASCADE,
    UNIQUE(user_id,community_id)
);
