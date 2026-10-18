create table if not exists pixels (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    site_id uuid not null references sites (id) on delete cascade,
    slug text not null unique,
    name text,
    enabled boolean not null default true,
    expires_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists pixels_site_idx on pixels (site_id);
