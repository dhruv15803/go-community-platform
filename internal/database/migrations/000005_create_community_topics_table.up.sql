


CREATE TABLE IF NOT EXISTS community_topics(
    community_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    FOREIGN KEY(community_id) REFERENCES communities(id) ON DELETE CASCADE,
    FOREIGN KEY(topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(community_id,topic_id)
);
