-- 记录从 User-Agent 解析出的设备品牌，例如 Apple、Samsung、Xiaomi。
alter table events add column if not exists device_brand text;
