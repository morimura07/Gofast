-- name: Ping :one
select 1;

-- Users --

-- name: SelectAllUsers :many
select * from users;

-- name: SelectUserByID :one
select * from users where id = $1;

-- name: SelectUserByCustomerID :one
select * from users where customer_id = $1;

-- name: SelectUserByEmailAndSub :one
select * from users where email = $1 and sub = $2;

-- name: InsertUser :one
insert into users (email, access, sub, avatar, api_key) values ($1, $2, $3, $4, $5) returning *;

-- name: UpdateUserActivity :exec
update users set updated = current_timestamp where id = $1;

-- name: UpdateUserAccess :one
update users set access = $1 where id = $2 returning *;

-- name: UpdateUserPhone :exec
update users set phone = $2 where id = $1;

-- name: UpdateUserCustomerID :exec
update users set customer_id = $1 where id = $2;

-- Auth Tokens --

-- name: SelectAuthTokenByID :one
select * from auth_tokens where id = $1;

-- name: InsertAuthToken :one
insert into auth_tokens (id, expires, user_id, provider, verifier, return_url) values ($1, $2, $3, $4, $5, $6) returning *;

-- name: UpdateAuthToken :exec
update auth_tokens set expires = $1 where id = $2;

-- name: DeleteExpiredAuthTokens :exec
delete from auth_tokens where expires < current_timestamp;

-- Skeletons --

-- name: SelectAllSkeletons :many
select * from skeletons where user_id = $1 order by created desc;

-- name: SelectSkeletonByID :one
select * from skeletons where id = $1 and user_id = $2;

-- name: InsertSkeleton :one
insert into skeletons (user_id, name, age, death, zombie) values ($1, $2, $3, $4, $5) returning *;

-- name: UpdateSkeleton :one
update skeletons set 
    name = $1,
    age = $2,
    death = $3,
    zombie = $4,
    updated = current_timestamp
where id = $5 and user_id = $6 returning *;

-- name: DeleteSkeleton :exec
delete from skeletons where id = $1 and user_id = $2;



