alter table sites
	drop column if exists filter_bots,
	drop column if exists exclude_cidrs;
