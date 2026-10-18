create table if not exists sites (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    name text not null,
    domain text not null default '',
    token text not null unique,
    share_token text not null unique,
    share_password_hash text,
    timezone text not null default 'UTC',
    created_at timestamptz not null default now()
);

create index if not exists sites_workspace_idx on sites (workspace_id);

-- 原始事件按月分区，由后台任务提前创建分区
create table if not exists events (
    id bigserial,
    site_id uuid not null references sites (id) on delete cascade,
    created_at timestamptz not null,
    type text not null check (type in ('pageview', 'custom', 'ping')),
    name text,
    visitor_id text not null,
    session_id text not null,
    url text,
    path text,
    title text,
    referrer text,
    referrer_host text,
    utm_source text,
    utm_medium text,
    utm_campaign text,
    channel text,
    browser text,
    os text,
    device text,
    screen text,
    language text,
    country text,
    props jsonb,
    primary key (id, created_at)
) partition by range (created_at);

create index if not exists events_site_created_idx on events (site_id, created_at);
create index if not exists events_site_path_idx on events (site_id, path, created_at);
create index if not exists events_site_visitor_idx on events (site_id, visitor_id, created_at);
create index if not exists events_site_session_idx on events (site_id, session_id);
create index if not exists events_site_type_name_idx on events (site_id, type, name, created_at);

create table if not exists sessions_web (
    site_id uuid not null references sites (id) on delete cascade,
    session_id text not null,
    visitor_id text not null,
    started_at timestamptz not null,
    last_seen_at timestamptz not null,
    pageviews integer not null default 0,
    entry_path text,
    exit_path text,
    primary key (site_id, session_id)
);

create index if not exists sessions_web_site_started_idx on sessions_web (site_id, started_at);
create index if not exists sessions_web_site_visitor_idx on sessions_web (site_id, visitor_id);
create index if not exists sessions_web_site_last_seen_idx on sessions_web (site_id, last_seen_at);

-- 每日聚合表：为后续长周期查询预留，原始事件永久保留
create table if not exists site_stats_daily (
    site_id uuid not null references sites (id) on delete cascade,
    day date not null,
    pv bigint not null default 0,
    uv bigint not null default 0,
    sessions bigint not null default 0,
    bounces bigint not null default 0,
    duration_sum bigint not null default 0,
    primary key (site_id, day)
);
