<script lang="ts">
	import type { EChartsCoreOption } from 'echarts/core';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { OverviewResponse } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import {
		averageDuration,
		formatChartTime,
		formatDuration,
		formatNumber,
		formatPercent
	} from '#lib/utils/format.js';

	let {
		workspaceId,
		siteId,
		range,
		timezone
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
		timezone: string;
	} = $props();

	let data = $state<OverviewResponse | null>(null);
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
			.overview(ws, site, span)
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

	const option = $derived<EChartsCoreOption>({
		grid: { left: 44, right: 16, top: 34, bottom: 30 },
		tooltip: { trigger: 'axis' },
		legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 11 } },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (data?.series ?? []).map((point) =>
				formatChartTime(point.bucket, timezone, data?.range.interval ?? 'hour')
			),
			axisLine: { lineStyle: { opacity: 0.25 } },
			axisLabel: { fontSize: 11, hideOverlap: true }
		},
		yAxis: {
			type: 'value',
			splitLine: { lineStyle: { opacity: 0.12 } },
			axisLabel: { fontSize: 11 }
		},
		series: [
			{
				name: 'PV',
				type: 'line',
				smooth: true,
				showSymbol: true,
				symbolSize: 5,
				areaStyle: { opacity: 0.1 },
				data: (data?.series ?? []).map((point) => point.pv)
			},
			{
				name: 'UV',
				type: 'line',
				smooth: true,
				showSymbol: true,
				symbolSize: 5,
				data: (data?.series ?? []).map((point) => point.uv)
			}
		]
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="stats-grid">
	<StatCard label="浏览量 PV" value={formatNumber(data?.overview.pv)} />
	<StatCard label="访客 UV" value={formatNumber(data?.overview.uv)} />
	<StatCard label="会话数" value={formatNumber(data?.overview.sessions)} />
	<StatCard
		label="跳出率"
		value={formatPercent(data?.overview.bounces ?? 0, data?.overview.sessions ?? 0)}
	/>
	<StatCard
		label="平均停留"
		value={formatDuration(
			averageDuration(data?.overview.duration_sum ?? 0, data?.overview.sessions ?? 0)
		)}
	/>
</div>

<div class="card">
	<div class="card-head">
		<h2>访问趋势</h2>
		{#if loading}
			<span class="small muted">加载中…</span>
		{/if}
	</div>

	{#if data && data.series.length > 0}
		<Chart {option} height="300px" />
	{:else}
		<p class="empty">暂无数据</p>
	{/if}
</div>
