<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { EChartsCoreOption } from 'echarts/core';
	import { ApiError } from '#lib/api/client.js';
	import { monitorApi } from '#lib/api/monitors.js';
	import type { Monitor, MonitorCheck, MonitorStatsResponse } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import RangePicker from '#lib/components/RangePicker.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import { session } from '#lib/stores/session.svelte.js';
	import { formatNumber } from '#lib/utils/format.js';

	const slug = $derived(page.params.ws ?? '');
	const monitorId = $derived(page.params.monitorId ?? '');
	const workspace = $derived(session.findWorkspace(slug));

	let monitor = $state<Monitor | null>(null);
	let data = $state<MonitorStatsResponse | null>(null);
	let checks = $state<MonitorCheck[]>([]);
	let range = $state('24h');
	let error = $state('');
	let notice = $state('');

	$effect(() => {
		if (!workspace || !monitorId) return;
		const wsId = workspace.id;
		let cancelled = false;

		monitorApi
			.get(wsId, monitorId)
			.then((result) => {
				if (!cancelled) monitor = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载监控失败';
			});

		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		if (!workspace || !monitorId) return;
		const wsId = workspace.id;
		const span = range;
		let cancelled = false;

		Promise.all([monitorApi.stats(wsId, monitorId, span), monitorApi.checks(wsId, monitorId, span)])
			.then(([stats, checkList]) => {
				if (cancelled) return;
				data = stats;
				checks = checkList.checks;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载统计失败';
			});

		return () => {
			cancelled = true;
		};
	});

	const option = $derived<EChartsCoreOption>({
		grid: { left: 48, right: 16, top: 34, bottom: 30 },
		tooltip: { trigger: 'axis' },
		legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 11 } },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (data?.stats.series ?? []).map((point) =>
				new Date(point.bucket).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit' })
			),
			axisLabel: { fontSize: 10, hideOverlap: true }
		},
		yAxis: { type: 'value', axisLabel: { fontSize: 11 }, splitLine: { lineStyle: { opacity: 0.12 } } },
		series: [
			{ name: 'P50', type: 'line', smooth: true, showSymbol: false, data: (data?.stats.series ?? []).map((p) => p.p50) },
			{ name: 'P95', type: 'line', smooth: true, showSymbol: false, data: (data?.stats.series ?? []).map((p) => p.p95) }
		]
	});

	async function mute() {
		if (!workspace || !monitor) return;
		try {
			await monitorApi.mute(workspace.id, monitor.id, 60);
			notice = '已静默 60 分钟';
		} catch (err) {
			error = err instanceof ApiError ? err.message : '静默失败';
		}
	}

	async function remove() {
		if (!workspace || !monitor) return;
		if (!confirm(`确定删除监控「${monitor.name}」？历史记录会一并删除。`)) return;
		try {
			await monitorApi.remove(workspace.id, monitor.id);
			await goto(`/app/w/${slug}/monitors`);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除失败';
		}
	}
</script>

<div class="page-head">
	<div>
		<h1>{monitor?.name ?? '监控详情'}</h1>
		<p class="small muted"><a href={`/app/w/${slug}/monitors`}>返回监控列表</a></p>
	</div>
	<div class="row">
		<RangePicker bind:value={range} />
		<button class="btn btn-sm" type="button" onclick={mute}>静默 60 分钟</button>
		<button class="btn btn-sm btn-danger" type="button" onclick={remove}>删除</button>
	</div>
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

<div class="stats-grid">
	<StatCard label="可用率" value={`${(data?.stats.availability ?? 0).toFixed(2)}%`} />
	<StatCard label="检测次数" value={formatNumber(data?.stats.checks)} />
	<StatCard label="失败次数" value={formatNumber(data?.stats.down)} />
	<StatCard label="P50" value={`${Math.round(data?.stats.p50 ?? 0)} ms`} />
	<StatCard label="P95" value={`${Math.round(data?.stats.p95 ?? 0)} ms`} />
</div>

<div class="card">
	<div class="card-head"><h2>响应时间趋势</h2></div>
	{#if data && data.stats.series.length > 0}
		<Chart {option} height="280px" />
	{:else}
		<p class="empty">暂无数据</p>
	{/if}
</div>

<div class="card" style="margin-top: 16px; overflow-x: auto">
	<div class="card-head"><h2>最近检测</h2></div>
	{#if checks.length === 0}
		<p class="empty">暂无记录</p>
	{:else}
		<table class="table">
			<thead><tr><th>时间</th><th>状态</th><th class="num">延迟</th><th class="num">状态码</th><th>错误</th></tr></thead>
			<tbody>
				{#each checks as check (check.id)}
					<tr>
						<td class="small muted">{new Date(check.checked_at).toLocaleString('zh-CN')}</td>
						<td><span class="badge" class:ok={check.status === 'up'}>{check.status === 'up' ? '正常' : '故障'}</span></td>
						<td class="num">{check.latency_ms} ms</td>
						<td class="num">{check.status_code || '—'}</td>
						<td class="small muted">{check.error || '—'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>
