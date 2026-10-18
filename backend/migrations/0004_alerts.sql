alter table monitors add column if not exists alert_after integer not null default 2;

alter table alert_states add column if not exists ssl_notified_at timestamptz;
alter table alert_states add column if not exists updated_at timestamptz not null default now();
