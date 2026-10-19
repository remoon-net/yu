<script lang="ts">
	import { DateTime } from 'luxon';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { RetentionResponse, StatsFilter } from '#lib/api/types.js';
	import { formatNumber } from '#lib/utils/format.js';

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

	let interval = $state<'day' | 'week'>('day');
	let data = $state<RetentionResponse | null>(null);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeInterval = interval;
		const activeFilter = filter;
		let cancelled = false;

		loading = true;
		error = '';

		analyticsApi
			.retention(ws, site, span, activeInterval, activeFilter)
			.then((result) => {
				if (!cancelled) data = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载留存失败';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});

	const columns = $derived(
		Math.max(0, ...(data?.cohorts ?? []).map((cohort) => cohort.cells.at(-1)?.period ?? 0))
	);

	function cohortLabel(date: string): string {
		const dt = DateTime.fromISO(date);
		if (!dt.isValid) return date;
		return dt.toFormat(interval === 'day' ? 'LL-dd' : "LL-dd '周'");
	}

	function heatStyle(rate: number): string {
		const alpha = 0.06 + Math.min(100, Math.max(0, rate)) / 100 * 0.54;
		return `background: rgba(59, 130, 246, ${alpha.toFixed(2)})`;
	}

	function periodLabel(period: number): string {
		if (period === 0) return interval === 'day' ? '首日' : '首周';
		return interval === 'day' ? `第 ${period} 天` : `第 ${period} 周`;
	}
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="card">
	<div class="card-head">
		<h2>留存</h2>
		<div class="row">
			<div class="tabs" style="margin: 0">
				<button
					type="button"
					class="tab"
					class:active={interval === 'day'}
					onclick={() => (interval = 'day')}>按天</button
				>
				<button
					type="button"
					class="tab"
					class:active={interval === 'week'}
					onclick={() => (interval = 'week')}>按周</button
				>
			</div>
		</div>
	</div>

	<p class="small muted" style="margin: 0 0 12px">
		按首次访问时间分组，统计之后仍回访的访客比例。尚未发生的周期显示为空。
	</p>

	{#if loading}
		<p class="empty">加载中…</p>
	{:else if !data || data.cohorts.length === 0}
		<p class="empty">暂无数据</p>
	{:else}
		<div style="overflow-x: auto">
			<table class="table retention">
				<thead>
					<tr>
						<th>队列</th>
						<th class="num">访客</th>
						{#each Array(columns + 1) as _, period (period)}
							<th class="num">{periodLabel(period)}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each data.cohorts as cohort (cohort.cohort)}
						<tr>
							<td>{cohortLabel(cohort.cohort)}</td>
							<td class="num">{formatNumber(cohort.size)}</td>
							{#each Array(columns + 1) as _, period (period)}
								{@const cell = cohort.cells.find((item) => item.period === period)}
								{#if cell}
									<td
										class="num heat"
										style={heatStyle(cell.rate)}
										title={`${formatNumber(cell.visitors)} / ${formatNumber(cohort.size)} 访客`}
									>
										{cell.rate.toFixed(0)}%
									</td>
								{:else}
									<td class="num muted">—</td>
								{/if}
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	.retention th,
	.retention td {
		white-space: nowrap;
	}

	.retention td.heat {
		font-variant-numeric: tabular-nums;
	}
</style>
