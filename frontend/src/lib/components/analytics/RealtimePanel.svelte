<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { RealtimeStats } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import { formatNumber } from '#lib/utils/format.js';

	let {
		workspaceId,
		siteId
	}: {
		workspaceId: string;
		siteId: string;
	} = $props();

	let data = $state<RealtimeStats | null>(null);
	let error = $state('');
	let updatedAt = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		let cancelled = false;

		const load = () => {
			analyticsApi
				.realtime(ws, site)
				.then((result) => {
					if (cancelled) return;
					data = result;
					updatedAt = new Date().toLocaleTimeString('zh-CN');
				})
				.catch((err) => {
					if (!cancelled) error = err instanceof Error ? err.message : '加载失败';
				});
		};

		load();
		const timer = setInterval(load, 30000);

		return () => {
			cancelled = true;
			clearInterval(timer);
		};
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="stats-grid">
	<StatCard label="最近 30 分钟访客" value={formatNumber(data?.visitors)} />
	<StatCard label="活跃会话（5 分钟）" value={formatNumber(data?.sessions)} />
</div>

<div class="card">
	<div class="card-head">
		<h2>正在浏览的页面</h2>
		{#if updatedAt}
			<span class="small muted">更新于 {updatedAt} · 每 30 秒刷新</span>
		{/if}
	</div>
	<BarList items={data?.pages ?? []} emptyText="最近 30 分钟暂无访问" />
</div>
