-- +goose Up
-- create "skeletons" table
create table if not exists skeletons (
    id uuid primary key default uuidv7(),
    created timestamptz not null default current_timestamp,
    updated timestamptz not null default current_timestamp,
    user_id uuid not null references users(id) on delete cascade,
    name text not null,
    age numeric not null,
    death timestamptz not null,
    zombie boolean not null default false
);

-- +goose Down
drop table if exists skeletons;
