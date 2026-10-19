<script lang="ts">
	import type { EChartsCoreOption } from 'echarts/core';
	import type { RangeInfo, ShortLinkDetail } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';
	import Chart from '#lib/components/Chart.svelte';
	import RangePicker from '#lib/components/RangePicker.svelte';
	import { formatChartTime, formatDateTime, formatNumber } from '#lib/utils/format.js';

	let {
		stats,
		rangeInfo,
		timezone,
		range,
		loading = false,
		error = '',
		target = null,
		targetLabel = '目标',
		countLabel = '点击',
		onRangeChange
	}: {
		stats: ShortLinkDetail | null;
		rangeInfo: RangeInfo | null;
		timezone: string;
		range: string;
		loading?: boolean;
		error?: string;
		target?: { text: string; internal?: boolean } | null;
		targetLabel?: string;
		countLabel?: string;
		onRangeChange: (value: string) => void;
	} = $props();

	const detailRanges = [
		{ value: '7d', label: '近 7 天' },
		{ value: '30d', label: '近 30 天' },
		{ value: '90d', label: '近 90 天' }
	];

	const option = $derived<EChartsCoreOption>({
		grid: { left: 44, right: 16, top: 34, bottom: 28 },
		tooltip: { trigger: 'axis' },
		legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 11 } },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (stats?.series ?? []).map((point) =>
				formatChartTime(point.bucket, timezone, rangeInfo?.interval ?? 'day')
			),
			axisLine: { lineStyle: { opacity: 0.25 } },
			axisLabel: { fontSize: 11, hideOverlap: true }
		},
		yAxis: {
			type: 'value',
			minInterval: 1,
			splitLine: { lineStyle: { opacity: 0.12 } },
			axisLabel: { fontSize: 11 }
		},
		series: [
			{
				name: countLabel,
				type: 'line',
				smooth: true,
				showSymbol: (stats?.series.length ?? 0) <= 1,
				symbolSize: 6,
				areaStyle: { opacity: 0.16 },
				data: (stats?.series ?? []).map((point) => point.pv)
			},
			{
				name: '访客',
				type: 'line',
				smooth: true,
				showSymbol: (stats?.series.length ?? 0) <= 1,
				symbolSize: 6,
				data: (stats?.series ?? []).map((point) => point.uv)
			}
		]
	});
</script>

<div class="detail-top">
	<div class="detail-metrics">
		<div class="metric">
			<span class="small muted">{countLabel}</span>
			<strong>{formatNumber(stats?.clicks ?? 0)}</strong>
		</div>
		<div class="metric">
			<span class="small muted">独立访客</span>
			<strong>{formatNumber(stats?.visitors ?? 0)}</strong>
		</div>
		<div class="metric">
			<span class="small muted">最近{countLabel}</span>
			<strong class="small">
				{stats?.last_click ? formatDateTime(stats.last_click, timezone) : '—'}
			</strong>
		</div>
	</div>
	<div class="row" style="gap: 6px">
		<RangePicker presets={detailRanges} value={range} onchange={onRangeChange} />
	</div>
</div>

{#if target}
	<div class="detail-target small muted">
		<span>{targetLabel}：</span>
		<code title={target.text}>{target.text}</code>
		{#if typeof target.internal === 'boolean'}
			<span class="badge" class:ok={target.internal}>{target.internal ? '站内' : '站外'}</span>
		{/if}
	</div>
{/if}

{#if error}
	<div class="error" style="margin: 12px 0">{error}</div>
{:else if loading && !stats}
	<p class="empty">加载中…</p>
{:else if stats}
	<div class="detail-chart">
		<Chart {option} height="240px" />
	</div>
	{#if stats.series.length === 0}
		<p class="small muted">该区间内还没有记录。</p>
	{/if}

	<div class="detail-grid">
		<section class="detail-tile">
			<h3>来源</h3>
			<BarList items={stats.referrers} />
		</section>
		<section class="detail-tile">
			<h3>渠道</h3>
			<BarList items={stats.channels} />
		</section>
		<section class="detail-tile">
			<h3>UTM 活动</h3>
			<BarList items={stats.utm_campaign} emptyText="未带活动参数" />
		</section>
		<section class="detail-tile">
			<h3>设备</h3>
			<BarList items={stats.device} />
		</section>
		<section class="detail-tile">
			<h3>UTM 来源</h3>
			<BarList items={stats.utm_source} emptyText="未带来源参数" />
		</section>
		<section class="detail-tile">
			<h3>浏览器</h3>
			<BarList items={stats.browser} />
		</section>
	</div>

	<h3 class="recent-title">最近{countLabel}明细</h3>
	{#if stats.recent.length === 0}
		<p class="empty">暂无明细。</p>
	{:else}
		<div style="overflow-x: auto">
			<table class="table recent-table">
				<thead>
					<tr>
						<th>时间</th>
						<th>来源</th>
						<th>渠道</th>
						<th>UTM</th>
						<th>设备</th>
					</tr>
				</thead>
				<tbody>
					{#each stats.recent as item, index (index)}
						<tr>
							<td class="small muted">{formatDateTime(item.created_at, timezone)}</td>
							<td class="small">{item.referrer_host || '(direct)'}</td>
							<td class="small">{item.channel || '—'}</td>
							<td class="small">{item.utm_campaign || item.utm_source || '—'}</td>
							<td class="small">
								{[item.browser, item.os, item.device].filter(Boolean).join(' · ') || '—'}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	.detail-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}

	.detail-metrics {
		display: flex;
		gap: 26px;
		flex-wrap: wrap;
	}

	.metric {
		display: grid;
		gap: 2px;
	}

	.metric strong {
		font-size: 20px;
		line-height: 1.2;
	}

	.detail-target {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 12px;
		min-width: 0;
	}

	.detail-target code {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.detail-chart {
		margin-top: 10px;
	}

	.detail-grid {
		display: grid;
		gap: 12px;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		margin-top: 6px;
	}

	@media (max-width: 760px) {
		.detail-grid {
			grid-template-columns: 1fr;
		}
	}

	.detail-tile {
		padding: 12px 14px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--panel-2);
	}

	.detail-tile h3 {
		margin: 0 0 8px;
		font-size: 13px;
		font-weight: 600;
		color: var(--muted);
	}

	.recent-title {
		margin: 20px 0 8px;
		font-size: 14px;
	}

	.recent-table :global(td) {
		padding: 8px 12px;
	}

	.recent-table :global(th) {
		padding: 8px 12px;
		position: sticky;
		top: 0;
		z-index: 1;
	}
</style>
