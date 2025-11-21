CREATE DATABASE urlshortener;

USE urlshortener;

CREATE TABLE urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    long_url VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    clicks BIGINT DEFAULT 0
);

CREATE INDEX idx_short_code ON urls(short_code);