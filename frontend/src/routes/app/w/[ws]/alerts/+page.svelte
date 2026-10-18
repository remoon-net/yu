<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import { alertApi, monitorApi } from '#lib/api/monitors.js';
	import type { Channel, Incident, MaintenanceWindow, Monitor } from '#lib/api/types.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let channels = $state<Channel[]>([]);
	let incidents = $state<Incident[]>([]);
	let windows = $state<MaintenanceWindow[]>([]);
	let monitors = $state<Monitor[]>([]);
	let error = $state('');
	let notice = $state('');

	let channelType = $state('webhook');
	let channelName = $state('');
	let channelTarget = $state('');
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUser = $state('');
	let smtpPass = $state('');
	let smtpFrom = $state('');
	let smtpTo = $state('');

	let windowTitle = $state('');
	let windowStart = $state('');
	let windowEnd = $state('');
	let windowMonitors = $state<string[]>([]);

	$effect(() => {
		if (!workspace) return;
		void load(workspace.id);
	});

	async function load(workspaceId: string) {
		try {
			[channels, incidents, windows, monitors] = await Promise.all([
				alertApi.channels(workspaceId),
				alertApi.incidents(workspaceId),
				alertApi.maintenance(workspaceId),
				monitorApi.list(workspaceId)
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : '加载告警数据失败';
		}
	}

	async function createChannel(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		const config =
			channelType === 'smtp'
				? {
						host: smtpHost.trim(),
						port: Number(smtpPort),
						username: smtpUser.trim(),
						password: smtpPass,
						from: smtpFrom.trim(),
						to: smtpTo.split(',').map((v) => v.trim()).filter(Boolean)
				  }
				: { url: channelTarget };

		try {
			await alertApi.createChannel(workspace.id, { type: channelType, name: channelName, config });
			channelName = '';
			channelTarget = '';
			smtpHost = '';
			smtpUser = '';
			smtpPass = '';
			smtpFrom = '';
			smtpTo = '';
			notice = '渠道已创建';
			await load(workspace.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建渠道失败';
		}
	}

	function configSummary(channel: Channel) {
		if (channel.type === 'smtp') {
			const config = channel.config as { host?: string; port?: number; to?: string[] };
			return `${config.host || '环境变量 SMTP'}:${config.port || 587} → ${(config.to ?? []).join(', ')}`;
		}
		return String((channel.config as { url?: string }).url ?? '');
	}

	async function testChannel(channel: Channel) {
		if (!workspace) return;
		try {
			const result = await alertApi.testChannel(workspace.id, channel.id);
			notice = result.ok ? '测试通知已发送' : `发送失败：${result.error}`;
		} catch (err) {
			error = err instanceof ApiError ? err.message : '测试失败';
		}
	}

	async function deleteChannel(channel: Channel) {
		if (!workspace || !confirm(`删除渠道「${channel.name}」？`)) return;
		await alertApi.deleteChannel(workspace.id, channel.id);
		await load(workspace.id);
	}

	async function resolveIncident(incident: Incident) {
		if (!workspace) return;
		await alertApi.resolveIncident(workspace.id, incident.id);
		await load(workspace.id);
	}

	async function createWindow(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;
		try {
			await alertApi.createMaintenance(workspace.id, {
				title: windowTitle,
				monitor_ids: windowMonitors,
				starts_at: new Date(windowStart).toISOString(),
				ends_at: new Date(windowEnd).toISOString()
			});
			windowTitle = '';
			notice = '维护窗口已创建';
			await load(workspace.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建维护窗口失败';
		}
	}

	async function deleteWindow(item: MaintenanceWindow) {
		if (!workspace || !confirm(`删除维护窗口「${item.title}」？`)) return;
		await alertApi.deleteMaintenance(workspace.id, item.id);
		await load(workspace.id);
	}
</script>

<div class="page-head">
	<div>
		<h1>告警与事件</h1>
		<p class="small muted">通知渠道、故障事件与维护窗口。</p>
	</div>
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

<div class="stack">
	<div class="card">
		<div class="card-head"><h2>通知渠道</h2></div>
		{#if channels.length === 0}
			<p class="empty">尚未配置通知渠道</p>
		{:else}
			<table class="table">
				<thead><tr><th>名称</th><th>类型</th><th>配置</th><th class="num">操作</th></tr></thead>
				<tbody>
					{#each channels as channel (channel.id)}
						<tr>
							<td>{channel.name}</td>
							<td>{channel.type}</td>
							<td class="small muted">{configSummary(channel)}</td>
							<td class="num">
								<button class="btn btn-sm" type="button" onclick={() => testChannel(channel)}>测试</button>
								<button class="btn btn-sm btn-danger" type="button" onclick={() => deleteChannel(channel)}>删除</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}

		{#if manageable}
			<form onsubmit={createChannel} style="margin-top: 16px">
				<div class="grid-2">
					<label class="field">名称<input class="input" bind:value={channelName} required /></label>
					<label class="field">类型
						<select class="select" bind:value={channelType}>
							<option value="webhook">Webhook</option>
							<option value="smtp">邮件</option>
						</select>
					</label>
					{#if channelType === 'webhook'}
						<label class="field">Webhook 地址<input class="input" bind:value={channelTarget} required placeholder="https://hooks.example.com/yu" /></label>
					{:else}
						<label class="field">SMTP 主机<input class="input" bind:value={smtpHost} placeholder="smtp.example.com" /></label>
						<label class="field">端口<input class="input" type="number" bind:value={smtpPort} min={1} max={65535} /></label>
						<label class="field">用户名<input class="input" bind:value={smtpUser} placeholder="可选" autocomplete="off" /></label>
						<label class="field">密码 / 授权码<input class="input" type="password" bind:value={smtpPass} autocomplete="new-password" /></label>
						<label class="field">发件人<input class="input" bind:value={smtpFrom} placeholder="Yu &lt;no-reply@example.com&gt;" /></label>
						<label class="field">收件人（逗号分隔）<input class="input" bind:value={smtpTo} required placeholder="a@example.com, b@example.com" /></label>
					{/if}
				</div>
				<div class="row" style="margin-top: 12px"><button class="btn btn-primary" type="submit">添加渠道</button></div>
			</form>
		{/if}
	</div>

	<div class="card">
		<div class="card-head"><h2>故障事件</h2></div>
		{#if incidents.length === 0}
			<p class="empty">暂无事件</p>
		{:else}
			<table class="table">
				<thead><tr><th>事件</th><th>级别</th><th>开始时间</th><th>状态</th><th></th></tr></thead>
				<tbody>
					{#each incidents as incident (incident.id)}
						<tr>
							<td><div>{incident.title}</div><div class="small muted">{incident.body}</div></td>
							<td>{incident.severity}</td>
							<td class="small muted">{new Date(incident.started_at).toLocaleString('zh-CN')}</td>
							<td><span class="badge" class:ok={incident.resolved_at !== null}>{incident.resolved_at ? '已恢复' : '进行中'}</span></td>
							<td class="num">
								{#if !incident.resolved_at && manageable}
									<button class="btn btn-sm" type="button" onclick={() => resolveIncident(incident)}>标记恢复</button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<div class="card">
		<div class="card-head"><h2>维护窗口</h2></div>
		{#if windows.length === 0}
			<p class="empty">暂无维护窗口</p>
		{:else}
			<table class="table">
				<thead><tr><th>标题</th><th>时间</th><th>监控数</th><th class="num"></th></tr></thead>
				<tbody>
					{#each windows as item (item.id)}
						<tr>
							<td>{item.title}</td>
							<td class="small muted">{new Date(item.starts_at).toLocaleString('zh-CN')} → {new Date(item.ends_at).toLocaleString('zh-CN')}</td>
							<td>{item.monitor_ids.length || '全部'}</td>
							<td class="num">
								{#if manageable}<button class="btn btn-sm btn-danger" type="button" onclick={() => deleteWindow(item)}>删除</button>{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}

		{#if manageable}
			<form onsubmit={createWindow} style="margin-top: 16px">
				<div class="grid-2">
					<label class="field">标题<input class="input" bind:value={windowTitle} required /></label>
					<label class="field">开始<input class="input" type="datetime-local" bind:value={windowStart} required /></label>
					<label class="field">结束<input class="input" type="datetime-local" bind:value={windowEnd} required /></label>
					<label class="field">监控（不选表示全部）
						<select class="select" multiple bind:value={windowMonitors} style="min-height: 90px">
							{#each monitors as monitor (monitor.id)}<option value={monitor.id}>{monitor.name}</option>{/each}
						</select>
					</label>
				</div>
				<div class="row" style="margin-top: 12px"><button class="btn btn-primary" type="submit">创建维护窗口</button></div>
			</form>
		{/if}
	</div>
</div>
