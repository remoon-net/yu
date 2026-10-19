-- 记录 IP 地理信息，配合可选的 MaxMind GeoLite2 数据库使用。
alter table events add column if not exists region text;
alter table events add column if not exists city text;
