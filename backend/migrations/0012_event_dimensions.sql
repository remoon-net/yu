-- 补充 UTM 与客户端版本维度。
alter table events add column if not exists utm_term text;
alter table events add column if not exists utm_content text;
alter table events add column if not exists browser_version text;
alter table events add column if not exists os_version text;
