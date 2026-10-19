alter table sites
	add column if not exists filter_bots boolean not null default true,
	add column if not exists exclude_cidrs text[] not null default array[
		'10.0.0.0/8',
		'172.16.0.0/12',
		'192.168.0.0/16',
		'127.0.0.0/8',
		'169.254.0.0/16',
		'::1/128',
		'fc00::/7',
		'fe80::/10'
	];
