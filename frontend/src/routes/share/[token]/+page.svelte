<script lang="ts">
	import { page } from '$app/state';
	import type { EChartsCoreOption } from 'echarts/core';
	import { ApiError, api } from '#lib/api/client.js';
	import type { ShareResponse } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import { formatNumber } from '#lib/utils/format.js';

	const token = $derived(page.params.token ?? '');
	let data = $state<ShareResponse | null>(null);
	let error = $state('');
	let needPassword = $state(false);
	let password = $state('');
	let unlocking = $state(false);

	$effect(() => {
		void load();
	});

	async function load() {
		try {
			data = await api.get<ShareResponse>(`/api/public/share/${token}?range=24h`);
			needPassword = false;
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				needPassword = true;
				return;
			}
			error = err instanceof Error ? err.message : '加载失败';
		}
	}

	async function unlock(event: SubmitEvent) {
		event.preventDefault();
		unlocking = true;
		try {
			data = await api.get<ShareResponse>(
				`/api/public/share/${token}?range=24h&password=${encodeURIComponent(password)}`
			);
			needPassword = false;
		} catch (err) {
			error = err instanceof ApiError ? err.message : '密码错误';
		} finally {
			unlocking = false;
		}
	}

	const option = $derived<EChartsCoreOption>({
		grid: { left: 44, right: 16, top: 30, bottom: 28 },
		tooltip: { trigger: 'axis' },
		legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 11 } },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (data?.series ?? []).map((point) => new Date(point.bucket).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })),
			axisLabel: { fontSize: 10, hideOverlap: true }
		},
		yAxis: { type: 'value', axisLabel: { fontSize: 11 }, splitLine: { lineStyle: { opacity: 0.12 } } },
		series: [
			{ name: 'PV', type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.1 }, data: (data?.series ?? []).map((p) => p.pv) },
			{ name: 'UV', type: 'line', smooth: true, showSymbol: false, data: (data?.series ?? []).map((p) => p.uv) }
		]
	});
</script>

<div class="auth-wrap" style="align-items: flex-start; padding-top: 40px">
	<div style="width: 100%; max-width: 900px" class="stack">
		{#if needPassword}
			<form class="auth-card" onsubmit={unlock}>
				<h1>需要访问密码</h1>
				{#if error}<div class="error">{error}</div>{/if}
				<label class="field">密码<input class="input" type="password" bind:value={password} required /></label>
				<button class="btn btn-primary" type="submit" disabled={unlocking}>
					{unlocking ? '验证中…' : '查看'}
				</button>
			</form>
		{:else if error}
			<div class="error">{error}</div>
		{:else if data}
			<div>
				<h1>{data.site.name}</h1>
				<p class="small muted">{data.site.domain} · 最近 24 小时</p>
			</div>

			<div class="stats-grid">
				<StatCard label="浏览量 PV" value={formatNumber(data.overview.pv)} />
				<StatCard label="访客 UV" value={formatNumber(data.overview.uv)} />
				<StatCard label="会话数" value={formatNumber(data.overview.sessions)} />
				<StatCard label="跳出率" value={`${((data.overview.bounces / Math.max(1, data.overview.sessions)) * 100).toFixed(1)}%`} />
			</div>

			<div class="card"><Chart {option} height="280px" /></div>

			<div class="grid-2">
				<div class="card">
					<div class="card-head"><h2>热门页面</h2></div>
					<table class="table">
						<tbody>
							{#each data.pages.slice(0, 10) as item (item.path)}
								<tr><td>{item.path}</td><td class="num">{formatNumber(item.views)}</td></tr>
							{/each}
						</tbody>
					</table>
				</div>
				<div class="card">
					<div class="card-head"><h2>来源</h2></div>
					<table class="table">
						<tbody>
							{#each data.referrers.slice(0, 10) as item (item.value)}
								<tr><td>{item.value}</td><td class="num">{formatNumber(item.count)}</td></tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{:else}
			<p class="empty">加载中…</p>
		{/if}
	</div>
</div>
