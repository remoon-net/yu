<script lang="ts">
	import type { EChartsCoreOption } from 'echarts/core';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { OverviewResponse, StatsFilter } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import BarList from '#lib/components/BarList.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import { bounceRate, deltaOf, rateDelta } from '#lib/utils/analytics.js';
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
		timezone,
		filter
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
		timezone: string;
		filter?: StatsFilter;
	} = $props();

	let data = $state<OverviewResponse | null>(null);
	let error = $state('');
	let loading = $state(true);

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeFilter = filter;
		let cancelled = false;

		loading = true;
		error = '';

		analyticsApi
			.overview(ws, site, span, activeFilter)
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
	<StatCard
		label="浏览量 PV"
		value={formatNumber(data?.overview.pv)}
		delta={deltaOf(data?.overview.pv ?? 0, data?.previous.pv ?? 0)}
	/>
	<StatCard
		label="访客 UV"
		value={formatNumber(data?.overview.uv)}
		delta={deltaOf(data?.overview.uv ?? 0, data?.previous.uv ?? 0)}
	/>
	<StatCard
		label="新访客"
		value={formatNumber(data?.overview.new_visitors)}
		delta={deltaOf(data?.overview.new_visitors ?? 0, data?.previous.new_visitors ?? 0)}
	/>
	<StatCard
		label="回访访客"
		value={formatNumber(data?.overview.returning_visitors)}
		delta={deltaOf(
			data?.overview.returning_visitors ?? 0,
			data?.previous.returning_visitors ?? 0
		)}
	/>
	<StatCard
		label="会话数"
		value={formatNumber(data?.overview.sessions)}
		delta={deltaOf(data?.overview.sessions ?? 0, data?.previous.sessions ?? 0)}
	/>
	<StatCard
		label="跳出率"
		value={formatPercent(data?.overview.bounces ?? 0, data?.overview.sessions ?? 0)}
		delta={rateDelta(
			bounceRate(data?.overview.bounces ?? 0, data?.overview.sessions ?? 0),
			bounceRate(data?.previous.bounces ?? 0, data?.previous.sessions ?? 0)
		)}
		invert
	/>
	<StatCard
		label="平均停留"
		value={formatDuration(
			averageDuration(data?.overview.duration_sum ?? 0, data?.overview.sessions ?? 0)
		)}
		delta={rateDelta(
			averageDuration(data?.overview.duration_sum ?? 0, data?.overview.sessions ?? 0),
			averageDuration(data?.previous.duration_sum ?? 0, data?.previous.sessions ?? 0)
		)}
		invert
	/>
	<StatCard
		label="活跃用户"
		value={`${formatNumber(data?.overview.dau)} / ${formatNumber(data?.overview.wau)} / ${formatNumber(data?.overview.mau)}`}
		hint="日 / 周 / 月"
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

<div class="card" style="margin-top: 16px">
	<div class="card-head"><h2>会话时长分布</h2></div>
	<BarList items={data?.duration_buckets ?? []} emptyText="暂无数据" />
</div>
