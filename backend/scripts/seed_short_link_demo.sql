-- 短链追踪演示数据（仅用于本地预览效果，生产环境不要执行）
--
-- 用法：
--   docker exec -i yu-pg psql -U yu -d yu < backend/scripts/seed_short_link_demo.sql
--
-- 清理：
--   delete from events where type = 'link_click' and name like 'demo-%';
--   delete from short_links where slug like 'demo-%';

-- 1. 近 90 天可能用到的 events 分区
create table if not exists events_202607 partition of events for values from ('2026-07-01') to ('2026-08-01');
create table if not exists events_202608 partition of events for values from ('2026-08-01') to ('2026-09-01');
create table if not exists events_202609 partition of events for values from ('2026-09-01') to ('2026-10-01');
create table if not exists events_202610 partition of events for values from ('2026-10-01') to ('2026-11-01');

-- 2. 三条演示短链（绑定 example.com 站点；一条站内、两条站外）
insert into short_links (id, workspace_id, site_id, slug, target_url, title, enabled, expires_at, created_at)
select gen_random_uuid(), s.workspace_id, s.id, v.slug, v.target, v.title, true, null, now() - interval '95 days'
from sites s
cross join (values
	('demo-autumn', 'https://shop.example.net/autumn?utm_source=weibo&utm_medium=social&utm_campaign=autumn-2026', '双十一预热'),
	('demo-newsletter', 'https://example.com/pricing?utm_source=newsletter&utm_medium=email&utm_campaign=sep-digest', '九月订阅邮件'),
	('demo-launch', 'https://github.com/remoon/yu?utm_source=hn&utm_medium=referral&utm_campaign=launch', '新产品发布')
) as v(slug, target, title)
where s.domain = 'example.com'
on conflict (slug) do nothing;

-- 3. 造最近 90 天的点击（越近越多，形成可看的趋势）
with src (idx, host, channel, utm_source, utm_medium) as (
	values
		(1, 'weibo.com', 'social', 'weibo', 'social'),
		(2, 't.co', 'social', 'twitter', 'social'),
		(3, 'mp.weixin.qq.com', 'social', 'wechat', 'social'),
		(4, 'www.google.com', 'search', 'google', 'organic'),
		(5, 'news.ycombinator.com', 'referral', 'hn', 'referral'),
		(6, '', 'direct', '', ''),
		(7, 'mail.qq.com', 'email', 'newsletter', 'email'),
		(8, 'example.com', 'referral', 'blog', 'referral')
),
agents (idx, browser, os, device) as (
	values
		(1, 'Chrome', 'Windows', 'desktop'),
		(2, 'Chrome', 'macOS', 'desktop'),
		(3, 'Safari', 'iOS', 'mobile'),
		(4, 'Safari', 'macOS', 'desktop'),
		(5, 'WeChat', 'iOS', 'mobile'),
		(6, 'Chrome', 'Android', 'mobile'),
		(7, 'Edge', 'Windows', 'desktop'),
		(8, 'Firefox', 'Linux', 'desktop')
),
days as (select generate_series(0, 89) as d),
links as (
	select id, site_id, slug, title, target_url,
	       case slug
	           when 'demo-autumn' then 'autumn-2026'
	           when 'demo-newsletter' then 'sep-digest'
	           else 'launch'
	       end as campaign
	from short_links
	where slug like 'demo-%'
),
expand as (
	select l.*, dd.d,
	       greatest(0, round((5 + 16 * exp(-dd.d / 30.0)) * (0.4 + random()))::int) as n
	from links l
	cross join days dd
),
clicks as (
	select e.*, gs as i,
	       1 + floor(random() * 8)::int as src_idx,
	       1 + floor(random() * 8)::int as agent_idx
	from expand e
	cross join lateral generate_series(1, e.n) as gs
)
insert into events (
	site_id, created_at, type, name, visitor_id, session_id,
	url, path, title, referrer, referrer_host,
	utm_source, utm_medium, utm_campaign, channel,
	browser, os, device, language
)
select
	c.site_id,
	now() - make_interval(days => c.d)
	      - make_interval(hours => floor(random() * 14 + 8)::int, mins => floor(random() * 60)::int),
	'link_click',
	c.slug,
	'demo-v' || (1 + floor(random() * 120))::int,
	'demo-s' || (1 + floor(random() * 160))::int,
	c.target_url,
	'/s/' || c.slug,
	c.title,
	case when s.host = '' then '' else 'https://' || s.host || '/post/' || c.i end,
	s.host,
	s.utm_source,
	s.utm_medium,
	c.campaign,
	s.channel,
	a.browser,
	a.os,
	a.device,
	'zh-CN'
from clicks c
join src s on s.idx = c.src_idx
join agents a on a.idx = c.agent_idx;
