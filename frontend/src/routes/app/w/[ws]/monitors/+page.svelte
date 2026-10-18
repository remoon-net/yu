<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.js';
	import { monitorApi } from '#lib/api/monitors.js';
	import type { Monitor } from '#lib/api/types.js';
	import Modal from '#lib/components/Modal.svelte';
	import { canManage, session } from '#lib/stores/session.svelte.js';
	import { parseInterval, parseTimeout, formatInterval } from '#lib/utils/duration.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let monitors = $state<Monitor[]>([]);
	let error = $state('');
	let formError = $state('');
	let showForm = $state(false);
	let saving = $state(false);

	let name = $state('');
	let type = $state('http');
	let target = $state('');
	let port = $state(443);
	let recordType = $state('A');
	let interval = $state('1m');
	let timeout = $state('10s');
	let alertAfter = $state(2);
	let groupName = $state('');

	function buildConfig() {
		switch (type) {
			case 'http':
				return { url: target };
			case 'tcp':
				return { host: target, port: Number(port) };
			case 'ping':
				return { host: target };
			default:
				return { name: target, record_type: recordType };
		}
	}

	$effect(() => {
		if (!workspace) return;
		void load(workspace.id);
	});

	async function load(workspaceId: string) {
		try {
			monitors = await monitorApi.list(workspaceId);
		} catch (err) {
			error = err instanceof Error ? err.message : '加载监控失败';
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		const intervalSec = parseInterval(interval);
		const timeoutMs = parseTimeout(timeout);

		if (intervalSec === null || intervalSec < 10 || intervalSec > 86400) {
			formError = '间隔格式不正确，支持 30s、1m、1h、1d（最小 10s）';
			return;
		}
		if (timeoutMs === null || timeoutMs < 500 || timeoutMs > 60000) {
			formError = '超时格式不正确，支持 500ms、10s、1m（500ms - 60s）';
			return;
		}

		saving = true;
		formError = '';

		try {
			await monitorApi.create(workspace.id, {
				name,
				type,
				config: buildConfig(),
				interval_sec: intervalSec,
				timeout_ms: timeoutMs,
				alert_after: Number(alertAfter),
				group_name: groupName
			});
			name = '';
			target = '';
			showForm = false;
			await load(workspace.id);
		} catch (err) {
			formError = err instanceof ApiError ? err.message : '创建监控失败';
		} finally {
			saving = false;
		}
	}

	function label(monitor: Monitor) {
		if (monitor.last_status === 'up') return '正常';
		if (monitor.last_status === 'down') return '故障';
		return '待检测';
	}
</script>

<div class="page-head">
	<div>
		<h1>服务监控</h1>
		<p class="small muted">HTTP / TCP / Ping / DNS 周期探测。</p>
	</div>
	{#if manageable}
		<button class="btn btn-primary" type="button" onclick={() => { formError = ''; showForm = true; }}>
			添加监控
		</button>
	{/if}
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}

<Modal open={showForm} title="添加监控" onclose={() => (showForm = false)}>
	<form onsubmit={create}>
		<div class="grid-2">
			<label class="field">名称<input class="input" bind:value={name} required maxlength={80} /></label>
			<label class="field">类型
				<select class="select" bind:value={type}>
					<option value="http">HTTP</option>
					<option value="tcp">TCP</option>
					<option value="ping">Ping</option>
					<option value="dns">DNS</option>
				</select>
			</label>
			<label class="field">
				{type === 'http' ? 'URL' : type === 'dns' ? '域名' : '主机'}
				<input class="input" bind:value={target} required placeholder={type === 'http' ? 'https://example.com' : 'example.com'} />
			</label>
			{#if type === 'tcp'}
				<label class="field">端口<input class="input" type="number" bind:value={port} min={1} max={65535} /></label>
			{/if}
			{#if type === 'dns'}
				<label class="field">记录类型
					<select class="select" bind:value={recordType}>
						<option>A</option><option>AAAA</option><option>CNAME</option><option>MX</option><option>TXT</option><option>NS</option>
					</select>
				</label>
			{/if}
			<label class="field">分组<input class="input" bind:value={groupName} placeholder="例如：核心服务" /></label>
			<label class="field">间隔<input class="input" bind:value={interval} placeholder="60s / 1m / 1h" /></label>
			<label class="field">超时<input class="input" bind:value={timeout} placeholder="500ms / 10s" /></label>
			<label class="field">连续失败几次告警<input class="input" type="number" bind:value={alertAfter} min={1} max={10} /></label>
		</div>
		{#if formError}<div class="error" style="margin-top: 12px">{formError}</div>{/if}
		<div class="modal-actions">
			<button class="btn" type="button" onclick={() => (showForm = false)}>取消</button>
			<button class="btn btn-primary" type="submit" disabled={saving}>{saving ? '创建中…' : '创建监控'}</button>
		</div>
	</form>
</Modal>

{#if monitors.length === 0}
	<div class="card"><p class="empty">还没有监控项。</p></div>
{:else}
	<div class="card" style="overflow-x: auto">
		<table class="table">
			<thead><tr><th>监控</th><th>状态</th><th class="num">延迟</th><th>最近检测</th><th></th></tr></thead>
			<tbody>
				{#each monitors as monitor (monitor.id)}
					<tr>
						<td>
							<div>{monitor.name}</div>
							<div class="small muted">{monitor.type} · {monitor.group_name || '未分组'} · 每 {formatInterval(monitor.interval_sec)}</div>
						</td>
						<td>
							<span class="badge" class:ok={monitor.last_status === 'up'}>{label(monitor)}</span>
							{#if monitor.last_error}<div class="small muted" title={monitor.last_error}>{monitor.last_error.slice(0, 48)}</div>{/if}
						</td>
						<td class="num">{monitor.last_latency_ms ?? '—'} ms</td>
						<td class="muted small">{monitor.last_checked_at ? new Date(monitor.last_checked_at).toLocaleString('zh-CN') : '—'}</td>
						<td>
							<div class="row" style="justify-content: flex-end; flex-wrap: nowrap">
								<a class="btn btn-sm" href={`/app/w/${slug}/monitors/${monitor.id}/?edit=1`}>编辑</a>
								<a class="btn btn-sm" href={`/app/w/${slug}/monitors/${monitor.id}/`}>详情</a>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
