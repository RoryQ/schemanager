-- Example migration file. Replace this file with what you need
create table users
(
    id          bigserial primary key,
    first_name  text,
    last_name   text,
    email       text,
    provider    text,
    token       text,
    inserted_at timestamptz not null,
    updated_at  timestamptz not null
);