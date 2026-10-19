<script lang="ts">
	import type { EChartsCoreOption } from 'echarts/core';
	import type { WorkspaceOverviewResponse } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';
	import Chart from '#lib/components/Chart.svelte';
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
		data,
		loading,
		timezone
	}: {
		data: WorkspaceOverviewResponse | null;
		loading: boolean;
		timezone: string;
	} = $props();

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

	const empty = $derived(loading && !data);
</script>

<div class="stats-grid">
	<StatCard
		label="浏览量 PV"
		value={empty ? '—' : formatNumber(data?.overview.pv)}
		delta={data ? deltaOf(data.overview.pv, data.previous.pv) : null}
	/>
	<StatCard
		label="访客 UV"
		value={empty ? '—' : formatNumber(data?.overview.uv)}
		delta={data ? deltaOf(data.overview.uv, data.previous.uv) : null}
	/>
	<StatCard
		label="新访客"
		value={empty ? '—' : formatNumber(data?.overview.new_visitors)}
		delta={data ? deltaOf(data.overview.new_visitors, data.previous.new_visitors) : null}
	/>
	<StatCard
		label="回访访客"
		value={empty ? '—' : formatNumber(data?.overview.returning_visitors)}
		delta={data
			? deltaOf(data.overview.returning_visitors, data.previous.returning_visitors)
			: null}
	/>
	<StatCard
		label="会话数"
		value={empty ? '—' : formatNumber(data?.overview.sessions)}
		delta={data ? deltaOf(data.overview.sessions, data.previous.sessions) : null}
	/>
	<StatCard
		label="跳出率"
		value={empty
			? '—'
			: formatPercent(data?.overview.bounces ?? 0, data?.overview.sessions ?? 0)}
		delta={data
			? rateDelta(
					bounceRate(data.overview.bounces, data.overview.sessions),
					bounceRate(data.previous.bounces, data.previous.sessions)
				)
			: null}
		invert
	/>
	<StatCard
		label="平均停留"
		value={empty
			? '—'
			: formatDuration(
					averageDuration(data?.overview.duration_sum ?? 0, data?.overview.sessions ?? 0)
				)}
		delta={data
			? rateDelta(
					averageDuration(data.overview.duration_sum, data.overview.sessions),
					averageDuration(data.previous.duration_sum, data.previous.sessions)
				)
			: null}
		invert
	/>
	<StatCard
		label="活跃用户"
		value={empty
			? '—'
			: `${formatNumber(data?.overview.dau)} / ${formatNumber(data?.overview.wau)} / ${formatNumber(data?.overview.mau)}`}
		hint="日 / 周 / 月"
	/>
</div>

<div class="card">
	<div class="card-head">
		<h2>访问趋势</h2>
		{#if loading && data}
			<span class="small muted">加载中…</span>
		{/if}
	</div>

	{#if empty}
		<p class="empty">加载中…</p>
	{:else if data && data.series.length > 0}
		<Chart {option} height="300px" />
	{:else}
		<p class="empty">暂无数据</p>
	{/if}
</div>

<div class="card" style="margin-top: 16px">
	<div class="card-head"><h2>会话时长分布</h2></div>
	<BarList items={data?.duration_buckets ?? []} emptyText="暂无数据" />
</div>
