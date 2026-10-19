-- 转化目标：事件或页面访问。
create table if not exists site_goals (
    id uuid primary key,
    site_id uuid not null references sites (id) on delete cascade,
    name text not null,
    type text not null check (type in ('event', 'page')),
    match_value text not null,
    created_at timestamptz not null default now()
);

create index if not exists site_goals_site_idx on site_goals (site_id, created_at);
