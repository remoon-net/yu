create table if not exists monitors (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    name text not null,
    type text not null check (type in ('http', 'tcp', 'ping', 'dns')),
    config jsonb not null default '{}'::jsonb,
    interval_sec integer not null default 60,
    timeout_ms integer not null default 10000,
    retries integer not null default 0,
    group_name text,
    tags text[] not null default '{}',
    ssl_alert_days integer not null default 14,
    enabled boolean not null default true,
    next_run_at timestamptz not null default now(),
    last_status text,
    last_checked_at timestamptz,
    last_latency_ms integer,
    last_error text,
    created_at timestamptz not null default now()
);

create index if not exists monitors_workspace_idx on monitors (workspace_id);
create index if not exists monitors_due_idx on monitors (next_run_at) where enabled;

create table if not exists monitor_checks (
    id bigserial,
    monitor_id uuid not null references monitors (id) on delete cascade,
    checked_at timestamptz not null,
    status text not null check (status in ('up', 'down')),
    latency_ms integer,
    status_code integer,
    error text,
    cert_expires_at timestamptz,
    primary key (id, checked_at)
) partition by range (checked_at);

create index if not exists monitor_checks_monitor_idx on monitor_checks (monitor_id, checked_at);

create table if not exists monitor_stats_hourly (
    monitor_id uuid not null references monitors (id) on delete cascade,
    bucket timestamptz not null,
    total integer not null default 0,
    up_count integer not null default 0,
    p50 integer,
    p95 integer,
    p99 integer,
    primary key (monitor_id, bucket)
);

create table if not exists notification_channels (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    type text not null check (type in ('smtp', 'webhook')),
    name text not null,
    config jsonb not null default '{}'::jsonb,
    enabled boolean not null default true,
    created_at timestamptz not null default now()
);

create index if not exists notification_channels_workspace_idx on notification_channels (workspace_id);

create table if not exists alert_states (
    monitor_id uuid primary key references monitors (id) on delete cascade,
    state text not null default 'up' check (state in ('up', 'pending', 'down', 'recovering')),
    consecutive_failures integer not null default 0,
    consecutive_successes integer not null default 0,
    since timestamptz not null default now(),
    down_since timestamptz,
    last_notified_at timestamptz,
    muted_until timestamptz,
    incident_id uuid
);

create table if not exists incidents (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    monitor_id uuid references monitors (id) on delete set null,
    title text not null,
    body text,
    severity text not null default 'major' check (severity in ('minor', 'major', 'critical')),
    source text not null default 'manual' check (source in ('auto', 'manual')),
    started_at timestamptz not null default now(),
    resolved_at timestamptz
);

create index if not exists incidents_workspace_idx on incidents (workspace_id, started_at desc);

create table if not exists maintenance_windows (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    title text not null,
    monitor_ids uuid[] not null default '{}',
    starts_at timestamptz not null,
    ends_at timestamptz not null
);

create index if not exists maintenance_windows_workspace_idx on maintenance_windows (workspace_id, starts_at desc);

create table if not exists status_pages (
    id uuid primary key,
    workspace_id uuid not null references workspaces (id) on delete cascade,
    slug text not null unique,
    name text not null,
    description text,
    password_hash text,
    theme text not null default 'dark',
    created_at timestamptz not null default now()
);

create table if not exists status_page_items (
    status_page_id uuid not null references status_pages (id) on delete cascade,
    monitor_id uuid not null references monitors (id) on delete cascade,
    display_name text,
    group_name text,
    sort_order integer not null default 0,
    primary key (status_page_id, monitor_id)
);
