

CREATE TYPE user_role AS ENUM('user','admin');

CREATE TABLE IF NOT EXISTS users(
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    password TEXT NOT NULL,
    username TEXT UNIQUE,
    is_verified BOOLEAN DEFAULT FALSE,
    role user_role DEFAULT 'user',
    user_image TEXT,
    bio TEXT,
    location TEXT,
    date_of_birth DATE,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);
