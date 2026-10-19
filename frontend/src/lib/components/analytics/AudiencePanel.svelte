<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { AudienceResponse, StatsFilter } from '#lib/api/types.js';
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

	let data = $state<AudienceResponse | null>(null);
	let error = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeFilter = filter;
		let cancelled = false;

		error = '';

		analyticsApi
			.audience(ws, site, span, activeFilter)
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
		{ key: 'device', title: '设备类型' },
		{ key: 'device_brand', title: '手机品牌' },
		{ key: 'browser', title: '浏览器' },
		{ key: 'browser_version', title: '浏览器版本' },
		{ key: 'os', title: '操作系统' },
		{ key: 'os_version', title: '系统版本' },
		{ key: 'screen', title: '屏幕尺寸' },
		{ key: 'language', title: '语言' },
		{ key: 'country', title: '国家/地区' },
		{ key: 'region', title: '地区' },
		{ key: 'city', title: '城市' }
	] as const;

	const regionNames = new Intl.DisplayNames(['zh-CN'], { type: 'region' });

	function countryLabel(code: string): string {
		try {
			return regionNames.of(code) ?? code;
		} catch {
			return code;
		}
	}
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="grid-2">
	{#each sections as section (section.key)}
		{@const items =
			section.key === 'country'
				? (data?.country ?? []).map((item) => ({ ...item, value: countryLabel(item.value) }))
				: (data?.[section.key] ?? [])}
		<div class="card">
			<div class="card-head"><h2>{section.title}</h2></div>
			<BarList {items} emptyText="暂无数据" />
		</div>
	{/each}
</div>
