


CREATE TABLE IF NOT EXISTS communities(
    id SERIAL PRIMARY KEY,
    community_name TEXT UNIQUE NOT NULL,
    community_description TEXT,
    community_image TEXT,
    community_owner_id INTEGER NOT NULL,
    community_created_at TIMESTAMP DEFAULT NOW(),
    community_updated_at TIMESTAMP,
    FOREIGN KEY(community_owner_id) REFERENCES users(id) ON DELETE CASCADE
);