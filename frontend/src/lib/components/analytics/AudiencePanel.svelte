<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { AudienceResponse } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';

	let {
		workspaceId,
		siteId,
		range
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
	} = $props();

	let data = $state<AudienceResponse | null>(null);
	let error = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		let cancelled = false;

		error = '';

		analyticsApi
			.audience(ws, site, span)
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

	const sections = [
		{ key: 'browser', title: '浏览器' },
		{ key: 'os', title: '操作系统' },
		{ key: 'device', title: '设备类型' },
		{ key: 'screen', title: '屏幕尺寸' },
		{ key: 'language', title: '语言' },
		{ key: 'country', title: '国家/地区' }
	] as const;
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="grid-2">
	{#each sections as section (section.key)}
		<div class="card">
			<div class="card-head"><h2>{section.title}</h2></div>
			<BarList items={data?.[section.key] ?? []} emptyText="暂无数据" />
		</div>
	{/each}
</div>
