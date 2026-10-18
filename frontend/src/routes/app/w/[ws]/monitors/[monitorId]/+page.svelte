<script lang="ts">
	import { goto, replaceState } from '$app/navigation';
	import { page } from '$app/state';
	import type { EChartsCoreOption } from 'echarts/core';
	import { ApiError } from '#lib/api/client.js';
	import { monitorApi } from '#lib/api/monitors.js';
	import type { Monitor, MonitorCheck, MonitorStatsResponse } from '#lib/api/types.js';
	import Chart from '#lib/components/Chart.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import RangePicker from '#lib/components/RangePicker.svelte';
	import StatCard from '#lib/components/StatCard.svelte';
	import { session } from '#lib/stores/session.svelte.js';
	import { formatInterval, formatTimeout, parseInterval, parseTimeout } from '#lib/utils/duration.js';
	import { formatNumber } from '#lib/utils/format.js';

	const slug = $derived(page.params.ws ?? '');
	const monitorId = $derived(page.params.monitorId ?? '');
	const workspace = $derived(session.findWorkspace(slug));

	let monitor = $state<Monitor | null>(null);
	let data = $state<MonitorStatsResponse | null>(null);
	let checks = $state<MonitorCheck[]>([]);
	let range = $state('24h');
	let error = $state('');
	let editError = $state('');
	let notice = $state('');

	let showEdit = $state(false);
	let autoOpened = false;
	let saving = $state(false);
	let editName = $state('');
	let editTarget = $state('');
	let editPort = $state(443);
	let editRecordType = $state('A');
	let editInterval = $state('1m');
	let editTimeout = $state('10s');
	let editRetries = $state(0);
	let editAlertAfter = $state(2);
	let editGroup = $state('');
	let editSSLDays = $state(14);
	let editEnabled = $state(true);

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

	$effect(() => {
		if (!autoOpened && monitor && page.url.searchParams.get('edit') === '1') {
			autoOpened = true;
			openEdit();
			replaceState(page.url.pathname, page.state);
		}
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

	function openEdit() {
		if (!monitor) return;
		editError = '';

		const config = monitor.config as Record<string, unknown>;
		editName = monitor.name;
		editInterval = formatInterval(monitor.interval_sec);
		editTimeout = formatTimeout(monitor.timeout_ms);
		editRetries = monitor.retries;
		editAlertAfter = monitor.alert_after;
		editGroup = monitor.group_name;
		editSSLDays = monitor.ssl_alert_days;
		editEnabled = monitor.enabled;

		if (monitor.type === 'http') {
			editTarget = String(config.url ?? '');
		} else if (monitor.type === 'tcp') {
			editTarget = String(config.host ?? '');
			editPort = Number(config.port ?? 443);
		} else if (monitor.type === 'ping') {
			editTarget = String(config.host ?? '');
		} else {
			editTarget = String(config.name ?? '');
			editRecordType = String(config.record_type ?? 'A');
		}

		showEdit = true;
	}

	async function saveEdit(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace || !monitor) return;

		const intervalSec = parseInterval(editInterval);
		const timeoutMs = parseTimeout(editTimeout);

		if (intervalSec === null || intervalSec < 10 || intervalSec > 86400) {
			editError = '间隔格式不正确，支持 30s、1m、1h、1d（最小 10s）';
			return;
		}
		if (timeoutMs === null || timeoutMs < 500 || timeoutMs > 60000) {
			editError = '超时格式不正确，支持 500ms、10s、1m（500ms - 60s）';
			return;
		}

		editError = '';
		const config = { ...(monitor.config as Record<string, unknown>) };
		if (monitor.type === 'http') config.url = editTarget.trim();
		if (monitor.type === 'tcp') {
			config.host = editTarget.trim();
			config.port = Number(editPort);
		}
		if (monitor.type === 'ping') config.host = editTarget.trim();
		if (monitor.type === 'dns') {
			config.name = editTarget.trim();
			config.record_type = editRecordType;
		}

		saving = true;
		error = '';

		try {
			monitor = await monitorApi.update(workspace.id, monitor.id, {
				name: editName,
				type: monitor.type,
				config,
				interval_sec: intervalSec,
				timeout_ms: timeoutMs,
				retries: Number(editRetries),
				alert_after: Number(editAlertAfter),
				group_name: editGroup,
				ssl_alert_days: Number(editSSLDays),
				enabled: editEnabled
			});
			showEdit = false;
			notice = '已保存';
		} catch (err) {
			editError = err instanceof ApiError ? err.message : '保存失败';
		} finally {
			saving = false;
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
		<button class="btn btn-sm" type="button" onclick={openEdit}>编辑</button>
		<button class="btn btn-sm" type="button" onclick={mute}>静默 60 分钟</button>
		<button class="btn btn-sm btn-danger" type="button" onclick={remove}>删除</button>
	</div>
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

{#if monitor}
	<Modal open={showEdit} title="编辑监控" onclose={() => (showEdit = false)}>
		<form onsubmit={saveEdit}>
			<div class="grid-2">
				<label class="field">名称<input class="input" bind:value={editName} required maxlength={80} /></label>
				<label class="field">分组<input class="input" bind:value={editGroup} placeholder="例如：核心服务" /></label>
				<label class="field">
					{monitor.type === 'http' ? 'URL' : monitor.type === 'dns' ? '域名' : '主机'}
					<input class="input" bind:value={editTarget} required />
				</label>
				{#if monitor.type === 'tcp'}
					<label class="field">端口<input class="input" type="number" bind:value={editPort} min={1} max={65535} /></label>
				{/if}
				{#if monitor.type === 'dns'}
					<label class="field">记录类型
						<select class="select" bind:value={editRecordType}>
							<option>A</option><option>AAAA</option><option>CNAME</option><option>MX</option><option>TXT</option><option>NS</option>
						</select>
					</label>
				{/if}
				<label class="field">间隔<input class="input" bind:value={editInterval} placeholder="60s / 1m / 1h" /></label>
				<label class="field">超时<input class="input" bind:value={editTimeout} placeholder="500ms / 10s" /></label>
				<label class="field">失败重试次数<input class="input" type="number" bind:value={editRetries} min={0} max={5} /></label>
				<label class="field">连续失败几次告警<input class="input" type="number" bind:value={editAlertAfter} min={1} max={10} /></label>
				{#if monitor.type === 'http'}
					<label class="field">证书到期提醒天数<input class="input" type="number" bind:value={editSSLDays} min={0} max={365} /></label>
				{/if}
			</div>
			<label class="row" style="margin-top: 14px; gap: 8px; font-size: 13px">
				<input type="checkbox" bind:checked={editEnabled} /> 启用该监控
			</label>
			{#if editError}<div class="error" style="margin-top: 12px">{editError}</div>{/if}
			<div class="modal-actions">
				<button class="btn" type="button" onclick={() => (showEdit = false)}>取消</button>
				<button class="btn btn-primary" type="submit" disabled={saving}>{saving ? '保存中…' : '保存'}</button>
			</div>
		</form>
	</Modal>
{/if}

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
