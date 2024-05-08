CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    gamertag text UNIQUE NOT NULL,
    password_hash bytea NOT NULL,
    platform text NOT NULL
);