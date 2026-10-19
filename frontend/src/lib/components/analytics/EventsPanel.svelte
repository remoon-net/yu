<script lang="ts">
	import type { EChartsCoreOption } from 'echarts/core';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { EventDetailResponse, EventsResponse, StatsFilter } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import { formatChartTime, formatDateTime, formatNumber } from '#lib/utils/format.js';

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

	let data = $state<EventsResponse | null>(null);
	let detail = $state<EventDetailResponse | null>(null);
	let selected = $state('');
	let error = $state('');
	let detailLoading = $state(false);

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeFilter = filter;
		let cancelled = false;

		error = '';
		detail = null;
		selected = '';

		analyticsApi
			.events(ws, site, span, activeFilter)
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

	async function selectEvent(name: string) {
		if (selected === name) {
			selected = '';
			detail = null;
			return;
		}

		selected = name;
		detail = null;
		detailLoading = true;

		try {
			detail = await analyticsApi.eventDetail(workspaceId, siteId, range, name, filter);
		} catch (err) {
			error = err instanceof Error ? err.message : '加载事件详情失败';
		} finally {
			detailLoading = false;
		}
	}

	const interval = $derived(range === '1h' || range === '24h' ? 'hour' : 'day');

	const option = $derived<EChartsCoreOption>({
		grid: { left: 44, right: 16, top: 24, bottom: 30 },
		tooltip: { trigger: 'axis' },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (detail?.series ?? []).map((point) =>
				formatChartTime(point.bucket, timezone, interval)
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
				name: '触发次数',
				type: 'line',
				smooth: true,
				showSymbol: false,
				areaStyle: { opacity: 0.1 },
				data: (detail?.series ?? []).map((point) => point.pv)
			}
		]
	});

	const propGroups = $derived.by(() => {
		const groups = new Map<string, { value: string; count: number }[]>();
		for (const item of detail?.props ?? []) {
			const list = groups.get(item.key) ?? [];
			list.push({ value: item.value, count: item.count });
			groups.set(item.key, list);
		}
		return [...groups.entries()];
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="card">
	<div class="card-head">
		<h2>自定义事件</h2>
		<span class="small muted">点击事件查看属性分布</span>
	</div>

	{#if data && data.events.length > 0}
		<div style="overflow-x: auto">
			<table class="table">
				<thead>
					<tr>
						<th>事件名</th>
						<th class="num">次数</th>
						<th class="num">访客</th>
						<th>最近触发</th>
					</tr>
				</thead>
				<tbody>
					{#each data.events as event (event.name)}
						<tr
							onclick={() => selectEvent(event.name)}
							style="cursor: pointer"
							style:background={selected === event.name ? 'var(--accent-soft)' : undefined}
						>
							<td>{event.name}</td>
							<td class="num">{formatNumber(event.count)}</td>
							<td class="num">{formatNumber(event.visitors)}</td>
							<td class="muted">{formatDateTime(event.last_seen, timezone)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="empty">暂无自定义事件</p>
	{/if}
</div>

{#if selected}
	<div class="card" style="margin-top: 16px">
		<div class="card-head">
			<h2>{selected}</h2>
			{#if detailLoading}
				<span class="small muted">加载中…</span>
			{/if}
		</div>

		{#if detail}
			{#if detail.series.length > 0}
				<Chart {option} height="260px" />
			{:else}
				<p class="empty">暂无趋势数据</p>
			{/if}

			{#if propGroups.length > 0}
				<h3 style="margin: 16px 0 10px">属性分布</h3>
				<div class="grid-2">
					{#each propGroups as [key, values] (key)}
						<div>
							<div class="small muted" style="margin-bottom: 6px">{key}</div>
							<ul class="bar-list">
								{#each values as item (item.value)}
									<li>
										<span class="bar-label" title={item.value}>{item.value}</span>
										<span class="bar-track"
											><span
												class="bar-fill"
												style="width: {(item.count / Math.max(1, ...values.map((v) => v.count))) *
													100}%"
											></span></span
										>
										<span class="bar-value">{formatNumber(item.count)}</span>
									</li>
								{/each}
							</ul>
						</div>
					{/each}
				</div>
			{:else}
				<p class="empty">该事件没有属性数据</p>
			{/if}
		{/if}
	</div>
{/if}
