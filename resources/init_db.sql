CREATE DATABASE urlshortener;

CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    long_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    clicks BIGINT NOT NULL DEFAULT 0
    );

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls (short_code);