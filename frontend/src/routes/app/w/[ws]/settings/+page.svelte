<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import type { APIKey, CreateInviteResponse, Invite, Member, Role } from '#lib/api/types.js';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';
	import { roleLabel } from '#lib/utils/roles.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));
	const isOwner = $derived(workspace?.role === 'owner');

	let members = $state<Member[]>([]);
	let invites = $state<Invite[]>([]);
	let error = $state('');
	let notice = $state('');
	let busy = $state(false);

	let inviteEmail = $state('');
	let inviteRole = $state<Role>('member');
	let inviteUrl = $state('');
	let inviteEmailSent = $state(false);

	let apiKeys = $state<APIKey[]>([]);
	let keyName = $state('');
	let newToken = $state('');
	let deletingWorkspace = $state(false);

	$effect(() => {
		if (!workspace) return;
		void loadAll(workspace.id);
	});

	async function loadAll(workspaceId: string) {
		error = '';

		try {
			const [memberList, inviteList, keyList] = await Promise.all([
				api.get<Member[]>(`/api/workspaces/${workspaceId}/members`),
				manageable
					? api.get<Invite[]>(`/api/workspaces/${workspaceId}/invites`)
					: Promise.resolve([] as Invite[]),
				manageable
					? api.get<APIKey[]>(`/api/workspaces/${workspaceId}/api-keys`)
					: Promise.resolve([] as APIKey[])
			]);

			members = memberList;
			invites = inviteList;
			apiKeys = keyList;
		} catch (err) {
			error = err instanceof Error ? err.message : '加载成员失败';
		}
	}

	async function createKey(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		try {
			const result = await api.post<{ key: APIKey; token: string }>(
				`/api/workspaces/${workspace.id}/api-keys`,
				{ name: keyName }
			);
			newToken = result.token;
			keyName = '';
			await loadAll(workspace.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建 API Key 失败';
		}
	}

	async function deleteKey(key: APIKey) {		if (!workspace) return;
		const ok = await confirmDialog({
			title: '删除 API Key',
			message: `确定删除 API Key「${key.name}」？使用该 Key 的集成会立即失效。`,
			confirmText: '删除'
		});
		if (!ok) return;
		await api.delete(`/api/workspaces/${workspace.id}/api-keys/${key.id}`);
		await loadAll(workspace.id);
	}

	async function invite(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		busy = true;
		error = '';
		notice = '';
		inviteUrl = '';

		try {
			const result = await api.post<CreateInviteResponse>(
				`/api/workspaces/${workspace.id}/invites`,
				{ email: inviteEmail, role: inviteRole }
			);

			inviteUrl = result.invite_url;
			inviteEmailSent = result.email_sent;
			notice = result.email_sent ? '邀请邮件已发送' : '邀请已创建，请复制链接发送给对方';
			inviteEmail = '';

			await loadAll(workspace.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建邀请失败';
		} finally {
			busy = false;
		}
	}

	async function updateRole(member: Member, role: string) {
		if (!workspace) return;

		error = '';
		notice = '';

		try {
			await api.patch(`/api/workspaces/${workspace.id}/members/${member.user_id}`, { role });
			await loadAll(workspace.id);
			notice = '成员角色已更新';
		} catch (err) {
			error = err instanceof ApiError ? err.message : '更新角色失败';
		}
	}

	async function deleteWorkspace() {
		if (!workspace) return;

		const ok = await confirmDialog({
			title: '删除工作区',
			message: `确定删除工作区「${workspace.name}」？\n站点、事件、监控、状态页、告警渠道与成员都会被删除，且无法恢复。`,
			confirmText: '删除工作区'
		});
		if (!ok) return;

		deletingWorkspace = true;
		error = '';

		try {
			await api.delete(`/api/workspaces/${workspace.id}`);
			await session.load(true);
			await goto('/app/');
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除工作区失败';
		} finally {
			deletingWorkspace = false;
		}
	}

	async function removeMember(member: Member) {
		if (!workspace) return;
		const ok = await confirmDialog({
			title: '移除成员',
			message: `确定将 ${member.email} 移出工作区？`,
			confirmText: '移除'
		});
		if (!ok) return;

		error = '';
		notice = '';

		try {
			await api.delete(`/api/workspaces/${workspace.id}/members/${member.user_id}`);
			await loadAll(workspace.id);
			notice = '成员已移除';
		} catch (err) {
			error = err instanceof ApiError ? err.message : '移除成员失败';
		}
	}

	async function deleteInvite(item: Invite) {
		if (!workspace) return;

		const ok = await confirmDialog({
			title: '撤销邀请',
			message: `确定撤销发给 ${item.email} 的邀请？该链接会立即失效。`,
			confirmText: '撤销'
		});
		if (!ok) return;

		error = '';
		notice = '';

		try {
			await api.delete(`/api/workspaces/${workspace.id}/invites/${item.id}`);
			await loadAll(workspace.id);
			notice = '邀请已撤销';
		} catch (err) {
			error = err instanceof ApiError ? err.message : '撤销邀请失败';
		}
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(inviteUrl);
			notice = '邀请链接已复制';
		} catch {
			error = '复制失败，请手动选择链接';
		}
	}
</script>

<div class="page-head">
	<div>
		<h1>工作区设置</h1>
		<p class="small muted">管理成员与邀请链接。</p>
	</div>
	<span class="badge">{roleLabel(workspace?.role)}</span>
</div>

{#if error}
	<div class="error" style="margin-bottom: 16px">{error}</div>
{/if}
{#if notice}
	<div class="badge ok" style="margin-bottom: 16px">{notice}</div>
{/if}

<div class="stack">
	<div class="card">
		<div class="card-head"><h2>成员</h2></div>

		<table class="table">
			<thead>
				<tr>
					<th>成员</th>
					<th>角色</th>
					<th class="num">操作</th>
				</tr>
			</thead>
			<tbody>
				{#each members as member (member.user_id)}
					<tr>
						<td>
							<div>{member.name}</div>
							<div class="small muted">{member.email}</div>
						</td>
						<td>
							{#if manageable}
								<select
									class="select"
									style="max-width: 130px"
									value={member.role}
									onchange={(event) => updateRole(member, event.currentTarget.value)}
								>
									{#if isOwner}<option value="owner">所有者</option>{/if}
									<option value="admin">管理员</option>
									<option value="member">成员</option>
								</select>
							{:else}
								{roleLabel(member.role)}
							{/if}
						</td>
						<td class="num">
							{#if manageable}
								<button
									class="btn btn-sm btn-danger"
									type="button"
									onclick={() => removeMember(member)}>移除</button
								>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if manageable && workspace}
		<form class="card" onsubmit={invite}>
			<div class="card-head"><h2>邀请成员</h2></div>

			<div class="grid-2">
				<label class="field">
					邮箱
					<input
						class="input"
						type="email"
						bind:value={inviteEmail}
						required
						placeholder="teammate@example.com"
					/>
				</label>

				<label class="field">
					角色
					<select class="select" bind:value={inviteRole}>
						{#if isOwner}<option value="owner">所有者</option>{/if}
						<option value="admin">管理员</option>
						<option value="member">成员</option>
					</select>
				</label>
			</div>

			<div class="row" style="margin-top: 14px">
				<button class="btn btn-primary" type="submit" disabled={busy}>
					{busy ? '创建中…' : '生成邀请'}
				</button>
				{#if !inviteEmailSent && inviteUrl}
					<span class="small muted">SMTP 未配置时将直接返回链接</span>
				{/if}
			</div>

			{#if inviteUrl}
				<div class="copy-box" style="margin-top: 14px">
					{inviteUrl}
					<button class="btn btn-sm" type="button" onclick={copy}>复制</button>
				</div>
			{/if}
		</form>

		<div class="card">
			<div class="card-head"><h2>待接受的邀请</h2></div>

			{#if invites.length === 0}
				<p class="empty">暂无待接受的邀请</p>
			{:else}
				<table class="table">
					<thead>
						<tr>
							<th>邮箱</th>
							<th>角色</th>
							<th>过期时间</th>
							<th class="num"></th>
						</tr>
					</thead>
					<tbody>
						{#each invites as item (item.id)}
							<tr>
								<td>{item.email}</td>
								<td>{roleLabel(item.role)}</td>
								<td class="muted">{new Date(item.expires_at).toLocaleString('zh-CN')}</td>
								<td class="num">
									{#if manageable}
										<button class="btn btn-sm btn-danger" type="button" onclick={() => deleteInvite(item)}>撤销</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>

		<div class="card">
			<div class="card-head"><h2>API Key</h2></div>

			{#if apiKeys.length === 0}
				<p class="empty">还没有 API Key</p>
			{:else}
				<table class="table">
					<thead><tr><th>名称</th><th>前缀</th><th>最近使用</th><th class="num"></th></tr></thead>
					<tbody>
						{#each apiKeys as key (key.id)}
							<tr>
								<td>{key.name}</td>
								<td class="small muted">{key.prefix}…</td>
								<td class="small muted">{key.last_used_at ? new Date(key.last_used_at).toLocaleString('zh-CN') : '从未使用'}</td>
								<td class="num"><button class="btn btn-sm btn-danger" type="button" onclick={() => deleteKey(key)}>删除</button></td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}

			<form onsubmit={createKey} style="margin-top: 16px">
				<div class="row">
					<input class="input" style="max-width: 260px" bind:value={keyName} required placeholder="Key 名称，例如 CI" />
					<button class="btn btn-primary" type="submit">创建</button>
				</div>
			</form>

			{#if newToken}
				<p class="small muted" style="margin-top: 12px">仅显示一次，请立即保存：</p>
				<div class="copy-box">{newToken}</div>
			{/if}

			<p class="small muted" style="margin-top: 12px">
				使用方式：<code>Authorization: Bearer &lt;token&gt;</code>
			</p>
		</div>

		{#if isOwner && workspace}
			<div class="card" style="border-color: color-mix(in srgb, var(--danger) 40%, var(--border))">
				<div class="card-head"><h2>危险操作</h2></div>
				<p class="small muted">
					删除工作区会一并删除其站点、事件、监控、状态页、告警渠道与成员，且无法恢复。
				</p>
				<button class="btn btn-danger" type="button" onclick={deleteWorkspace} disabled={deletingWorkspace}>
					{deletingWorkspace ? '删除中…' : '删除工作区'}
				</button>
			</div>
		{/if}
	{/if}
</div>
