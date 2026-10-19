<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { SourcesResponse, StatsFilter } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';

	let {
		workspaceId,
		siteId,
		range,
		filter
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
		filter?: StatsFilter;
	} = $props();

	let data = $state<SourcesResponse | null>(null);
	let error = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeFilter = filter;
		let cancelled = false;

		error = '';

		analyticsApi
			.sources(ws, site, span, activeFilter)
			.then((result) => {
				if (!cancelled) data = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载失败';
			});

		return () => {
			cancelled = true;
		};
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="grid-2">
	<div class="card">
		<div class="card-head"><h2>来源域名</h2></div>
		<BarList items={data?.referrers ?? []} emptyText="暂无来源数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>来源页面</h2></div>
		<BarList items={data?.referrer_urls ?? []} emptyText="暂无来源页面" />
	</div>

	<div class="card">
		<div class="card-head"><h2>渠道</h2></div>
		<BarList items={data?.channels ?? []} emptyText="暂无渠道数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>UTM Source</h2></div>
		<BarList items={data?.utm_source ?? []} emptyText="暂无 UTM 数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>UTM Medium</h2></div>
		<BarList items={data?.utm_medium ?? []} emptyText="暂无 UTM 数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>UTM Campaign</h2></div>
		<BarList items={data?.utm_campaign ?? []} emptyText="暂无 UTM 数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>UTM Term</h2></div>
		<BarList items={data?.utm_term ?? []} emptyText="暂无 UTM 数据" />
	</div>

	<div class="card">
		<div class="card-head"><h2>UTM Content</h2></div>
		<BarList items={data?.utm_content ?? []} emptyText="暂无 UTM 数据" />
	</div>
</div>
