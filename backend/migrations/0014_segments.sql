-- 保存的访客分段（过滤条件）。
create table if not exists site_segments (
    id uuid primary key,
    site_id uuid not null references sites (id) on delete cascade,
    name text not null,
    filters jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists site_segments_site_idx on site_segments (site_id, created_at);
