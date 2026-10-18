create table if not exists users (
    id uuid primary key,
    email text not null,
    password_hash text not null,
    name text not null,
    created_at timestamptz not null default now()
);

create unique index if not exists users_email_lower_idx on users (lower(email));

create table if not exists sessions_auth (
    id text primary key,
    user_id uuid not null references users (id) on delete cascade,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);

create index if not exists sessions_auth_user_idx on sessions_auth (user_id);
create index if not exists sessions_auth_expires_idx on sessions_auth (expires_at);

create table if not exists workspaces (
    id uuid primary key,
    name text not null,
    slug text not null unique,
    created_at timestamptz not null default now()
);

create table if not exists workspace_members (
    workspace_id uuid not null references workspaces (id) on delete cascade,
    user_id uuid not null references users (id) on delete cascade,
    role text not null check (role in ('owner', 'admin', 'member')),
    created_at timestamptz not null default now(),
    primary key (workspace_id, user_id)
);

create index if not exists workspace_members_user_idx on workspace_members (user_id);

create table if not exists invites (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    email text not null,
    role text not null check (role in ('owner', 'admin', 'member')),
    token_hash text not null unique,
    invited_by uuid references users (id) on delete set null,
    expires_at timestamptz not null,
    accepted_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists invites_workspace_idx on invites (workspace_id);
create index if not exists invites_email_idx on invites (lower(email));
