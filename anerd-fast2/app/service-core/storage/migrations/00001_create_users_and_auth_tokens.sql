-- +goose Up
-- create "users" table
create table if not exists users (
    id uuid primary key default uuidv7(),
    created timestamptz not null default current_timestamp,
    updated timestamptz not null default current_timestamp,
    email text not null,
    phone text not null default '',
    access bigint not null,
    sub text not null,
    avatar text not null default '',
    api_key text not null default '',
    customer_id text not null default '',
    unique (email, sub)
);

-- create "auth_tokens" table
create table if not exists auth_tokens (
    id text primary key not null,
    created timestamptz not null default current_timestamp,
    expires timestamptz not null,
    -- for refresh tokens
    user_id uuid references users(id) on delete cascade,
    -- for oauth state tokens
    provider text not null default '',
    verifier text not null default '',
    return_url text not null default ''
);

-- +goose Down
drop table if exists auth_tokens;
drop table if exists users;
