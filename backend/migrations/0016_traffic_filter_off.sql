alter table sites
	alter column filter_bots set default false,
	alter column exclude_cidrs set default '{}'::text[];

update sites
set filter_bots = false,
	exclude_cidrs = '{}'::text[];
