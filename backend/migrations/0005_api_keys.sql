create table if not exists api_keys (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    user_id uuid references users (id) on delete set null,
    name text not null,
    key_prefix text not null,
    key_hash text not null unique,
    created_at timestamptz not null default now(),
    last_used_at timestamptz
);

create index if not exists api_keys_workspace_idx on api_keys (workspace_id);
