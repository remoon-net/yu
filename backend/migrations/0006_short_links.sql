create table if not exists short_links (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    site_id uuid not null references sites (id) on delete cascade,
    slug text not null unique,
    target_url text not null,
    title text,
    enabled boolean not null default true,
    expires_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists short_links_site_idx on short_links (site_id);
