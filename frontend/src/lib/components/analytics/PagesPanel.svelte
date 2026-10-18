<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { PagesResponse } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';
	import { formatNumber } from '#lib/utils/format.js';

	let {
		workspaceId,
		siteId,
		range
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
	} = $props();

	let data = $state<PagesResponse | null>(null);
	let error = $state('');
	let loading = $state(true);

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		let cancelled = false;

		loading = true;
		error = '';

		analyticsApi
			.pages(ws, site, span)
			.then((result) => {
				if (!cancelled) data = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载失败';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="card">
	<div class="card-head">
		<h2>热门页面</h2>
		{#if loading}
			<span class="small muted">加载中…</span>
		{/if}
	</div>

	{#if data && data.pages.length > 0}
		<div style="overflow-x: auto">
			<table class="table">
				<thead>
					<tr>
						<th>页面</th>
						<th>标题</th>
						<th class="num">浏览量</th>
						<th class="num">访客</th>
					</tr>
				</thead>
				<tbody>
					{#each data.pages as page (page.path)}
						<tr>
							<td>{page.path}</td>
							<td class="muted">{page.title || '—'}</td>
							<td class="num">{formatNumber(page.views)}</td>
							<td class="num">{formatNumber(page.visitors)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="empty">暂无数据</p>
	{/if}
</div>

<div class="grid-2" style="margin-top: 16px">
	<div class="card">
		<div class="card-head"><h2>入口页</h2></div>
		<BarList items={data?.entry ?? []} />
	</div>

	<div class="card">
		<div class="card-head"><h2>退出页</h2></div>
		<BarList items={data?.exit ?? []} />
	</div>
</div>
