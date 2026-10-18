<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import { alertApi, monitorApi } from '#lib/api/monitors.js';
	import type { Channel, Incident, MaintenanceWindow, Monitor } from '#lib/api/types.js';
	import Modal from '#lib/components/Modal.svelte';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
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
	let showChannel = $state(false);
	let editingChannel = $state<Channel | null>(null);
	let channelError = $state('');
	let channelSaving = $state(false);
	let channelEnabled = $state(true);
	let showWindow = $state(false);

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

	function resetChannelForm() {
		channelType = 'webhook';
		channelName = '';
		channelTarget = '';
		smtpHost = '';
		smtpPort = 587;
		smtpUser = '';
		smtpPass = '';
		smtpFrom = '';
		smtpTo = '';
		channelEnabled = true;
		channelError = '';
	}

	function openCreateChannel() {
		editingChannel = null;
		resetChannelForm();
		showChannel = true;
	}

	function openEditChannel(channel: Channel) {
		editingChannel = channel;
		resetChannelForm();
		channelType = channel.type;
		channelName = channel.name;
		channelEnabled = channel.enabled;

		if (channel.type === 'smtp') {
			const config = channel.config as {
				host?: string;
				port?: number;
				username?: string;
				from?: string;
				to?: string[];
			};
			smtpHost = config.host ?? '';
			smtpPort = config.port ?? 587;
			smtpUser = config.username ?? '';
			smtpFrom = config.from ?? '';
			smtpTo = (config.to ?? []).join(', ');
		} else {
			channelTarget = String((channel.config as { url?: string }).url ?? '');
		}

		showChannel = true;
	}

	async function submitChannel(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		const baseConfig: Record<string, unknown> =
			channelType === 'smtp'
				? {
						host: smtpHost.trim(),
						port: Number(smtpPort),
						username: smtpUser.trim(),
						password: smtpPass,
						from: smtpFrom.trim(),
						to: smtpTo
							.split(',')
							.map((v) => v.trim())
							.filter(Boolean)
				  }
				: { url: channelTarget.trim() };

		// 编辑时密码留空则不修改原密码。
		if (
			editingChannel &&
			channelType === 'smtp' &&
			!smtpPass &&
			editingChannel.type === 'smtp'
		) {
			const original = editingChannel.config as { password?: string };
			baseConfig.password = original.password ?? '';
		}

		channelSaving = true;
		channelError = '';

		try {
			if (editingChannel) {
				await alertApi.updateChannel(workspace.id, editingChannel.id, {
					name: channelName,
					config: baseConfig,
					enabled: channelEnabled
				});
				notice = '渠道已更新';
			} else {
				await alertApi.createChannel(workspace.id, {
					type: channelType,
					name: channelName,
					config: baseConfig,
					enabled: channelEnabled
				});
				notice = '渠道已创建';
			}

			showChannel = false;
			editingChannel = null;
			resetChannelForm();
			await load(workspace.id);
		} catch (err) {
			channelError = err instanceof ApiError ? err.message : '保存渠道失败';
		} finally {
			channelSaving = false;
		}
	}

	function configSummary(channel: Channel) {
		if (channel.type === 'smtp') {
			const config = channel.config as { host?: string; port?: number; to?: string[] };
			return `${config.host || '未配置主机'}:${config.port || 587} → ${(config.to ?? []).join(', ')}`;
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
		if (!workspace) return;

		// 从编辑弹窗触发时，先关闭编辑弹窗再弹确认，取消可返回。
		const fromForm = editingChannel?.id === channel.id;
		if (fromForm) showChannel = false;

		const ok = await confirmDialog({
			title: '删除通知渠道',
			message: `确定删除渠道「${channel.name}」？删除后该渠道不再接收告警。`,
			confirmText: '删除'
		});

		if (!ok) {
			if (fromForm) showChannel = true;
			return;
		}

		await alertApi.deleteChannel(workspace.id, channel.id);
		editingChannel = null;
		resetChannelForm();
		notice = '渠道已删除';
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
			showWindow = false;
			notice = '维护窗口已创建';
			await load(workspace.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建维护窗口失败';
		}
	}

	async function deleteWindow(item: MaintenanceWindow) {
		if (!workspace) return;
		const ok = await confirmDialog({
			title: '删除维护窗口',
			message: `确定删除维护窗口「${item.title}」？`,
			confirmText: '删除'
		});
		if (!ok) return;
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
		<div class="card-head">
			<h2>通知渠道</h2>
			{#if manageable}<button class="btn btn-sm" type="button" onclick={openCreateChannel}>添加渠道</button>{/if}
		</div>
		{#if channels.length === 0}
			<p class="empty">尚未配置通知渠道</p>
		{:else}
			<table class="table">
				<thead><tr><th>名称</th><th>类型</th><th>配置</th><th>状态</th><th class="num">操作</th></tr></thead>
				<tbody>
					{#each channels as channel (channel.id)}
						<tr>
							<td>{channel.name}</td>
							<td>{channel.type}</td>
							<td class="small muted">{configSummary(channel)}</td>
							<td><span class="badge" class:ok={channel.enabled}>{channel.enabled ? '启用' : '停用'}</span></td>
							<td class="num">
								{#if manageable}
									<button class="btn btn-sm" type="button" onclick={() => openEditChannel(channel)}>编辑</button>
									<button class="btn btn-sm" type="button" onclick={() => testChannel(channel)}>测试</button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}

		{#if manageable}
			<Modal
				open={showChannel}
				title={editingChannel ? '编辑通知渠道' : '添加通知渠道'}
				onclose={() => (showChannel = false)}
			>
				<form onsubmit={submitChannel}>
					<div class="grid-2">
						<label class="field">名称<input class="input" bind:value={channelName} required maxlength={60} /></label>
						<label class="field">类型
							<select class="select" bind:value={channelType} disabled={!!editingChannel}>
								<option value="webhook">Webhook</option>
								<option value="smtp">邮件</option>
							</select>
						</label>
						{#if channelType === 'webhook'}
							<label class="field">Webhook 地址<input class="input" bind:value={channelTarget} required placeholder="https://hooks.example.com/yu" /></label>
						{:else}
							<label class="field">SMTP 主机<input class="input" bind:value={smtpHost} required placeholder="smtp.example.com" /></label>
							<label class="field">端口<input class="input" type="number" bind:value={smtpPort} min={1} max={65535} /></label>
							<label class="field">用户名<input class="input" bind:value={smtpUser} placeholder="可选" autocomplete="off" /></label>
							<label class="field">密码 / 授权码<input class="input" type="password" bind:value={smtpPass} autocomplete="new-password" placeholder={editingChannel ? '留空表示不修改' : ''} /></label>
							<label class="field">发件人<input class="input" bind:value={smtpFrom} placeholder="Yu &lt;no-reply@example.com&gt;" /></label>
							<label class="field">收件人（逗号分隔）<input class="input" bind:value={smtpTo} required placeholder="a@example.com, b@example.com" /></label>
						{/if}
					</div>
					<label class="row" style="margin-top: 14px; gap: 8px; font-size: 13px">
						<input type="checkbox" bind:checked={channelEnabled} /> 启用该渠道
					</label>
					{#if channelError}<div class="error" style="margin-top: 12px">{channelError}</div>{/if}
					<div class="modal-actions">
						{#if editingChannel}
							<button
								class="btn btn-danger"
								type="button"
								style="margin-right: auto"
								onclick={() => deleteChannel(editingChannel!)}
							>
								删除渠道
							</button>
						{/if}
						<button class="btn" type="button" onclick={() => (showChannel = false)}>取消</button>
						<button class="btn btn-primary" type="submit" disabled={channelSaving}>
							{channelSaving ? '保存中…' : editingChannel ? '保存' : '添加渠道'}
						</button>
					</div>
				</form>
			</Modal>
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
		<div class="card-head">
			<h2>维护窗口</h2>
			{#if manageable}<button class="btn btn-sm" type="button" onclick={() => (showWindow = true)}>创建维护窗口</button>{/if}
		</div>
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
			<Modal open={showWindow} title="创建维护窗口" onclose={() => (showWindow = false)}>
				<form onsubmit={createWindow}>
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
					<div class="modal-actions">
						<button class="btn" type="button" onclick={() => (showWindow = false)}>取消</button>
						<button class="btn btn-primary" type="submit">创建维护窗口</button>
					</div>
				</form>
			</Modal>
		{/if}
	</div>
</div>
