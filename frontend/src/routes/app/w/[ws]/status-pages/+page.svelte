<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.js';
	import { monitorApi, statusApi } from '#lib/api/monitors.js';
	import type { Monitor, StatusPage } from '#lib/api/types.js';
	import Modal from '#lib/components/Modal.svelte';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let pages = $state<StatusPage[]>([]);
	let monitors = $state<Monitor[]>([]);
	let error = $state('');
	let notice = $state('');

	let name = $state('');
	let pageSlug = $state('');
	let description = $state('');
	let password = $state('');
	let showCreate = $state(false);
	let createError = $state('');
	let creating = $state(false);

	let editing = $state('');
	let selected = $state<string[]>([]);
	let groups = $state<Record<string, string>>({});

	$effect(() => {
		if (!workspace) return;
		void load(workspace.id);
	});

	async function load(workspaceId: string) {
		try {
			[pages, monitors] = await Promise.all([statusApi.list(workspaceId), monitorApi.list(workspaceId)]);
		} catch (err) {
			error = err instanceof Error ? err.message : '加载状态页失败';
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		creating = true;
		createError = '';

		try {
			await statusApi.create(workspace.id, {
				slug: pageSlug,
				name,
				description,
				password,
				theme: 'dark'
			});
			name = '';
			pageSlug = '';
			description = '';
			password = '';
			showCreate = false;
			notice = '状态页已创建';
			await load(workspace.id);
		} catch (err) {
			createError = err instanceof ApiError ? err.message : '创建失败';
		} finally {
			creating = false;
		}
	}

	async function edit(pageItem: StatusPage) {
		if (!workspace) return;
		if (editing === pageItem.id) {
			editing = '';
			return;
		}
		const detail = await statusApi.get(workspace.id, pageItem.id);
		selected = detail.items.map((item) => item.monitor_id);
		groups = Object.fromEntries(detail.items.map((item) => [item.monitor_id, item.group_name]));
		editing = pageItem.id;
	}

	async function saveItems(pageItem: StatusPage) {
		if (!workspace) return;
		try {
			await statusApi.putItems(
				workspace.id,
				pageItem.id,
				selected.map((monitorId) => ({
					monitor_id: monitorId,
					group_name: groups[monitorId] ?? ''
				}))
			);
			notice = '组件已保存';
			editing = '';
		} catch (err) {
			error = err instanceof Error ? err.message : '保存失败';
		}
	}

	async function remove(pageItem: StatusPage) {
		if (!workspace) return;
		const ok = await confirmDialog({
			title: '删除状态页',
			message: `确定删除状态页「${pageItem.name}」？公开链接会立即失效。`,
			confirmText: '删除'
		});
		if (!ok) return;
		await statusApi.remove(workspace.id, pageItem.id);
		await load(workspace.id);
	}

	function origin() {
		return window.location.origin;
	}
</script>

<div class="page-head">
	<div>
		<h1>状态页</h1>
		<p class="small muted">对外展示服务可用性与历史事件。</p>
	</div>
	{#if manageable}
		<button class="btn btn-primary" type="button" onclick={() => { createError = ''; showCreate = true; }}>新建状态页</button>
	{/if}
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

{#if manageable}
	<Modal open={showCreate} title="新建状态页" onclose={() => (showCreate = false)}>
		<form onsubmit={create}>
			<div class="grid-2">
				<label class="field">名称<input class="input" bind:value={name} required maxlength={80} /></label>
				<label class="field">Slug<input class="input" bind:value={pageSlug} required pattern={"[a-z0-9][a-z0-9\\-]{0,48}[a-z0-9]"} placeholder="status" /></label>
				<label class="field">描述<input class="input" bind:value={description} /></label>
				<label class="field">访问密码（可选）<input class="input" type="password" bind:value={password} /></label>
			</div>
			{#if createError}<div class="error" style="margin-top: 12px">{createError}</div>{/if}
			<div class="modal-actions">
				<button class="btn" type="button" onclick={() => (showCreate = false)}>取消</button>
				<button class="btn btn-primary" type="submit" disabled={creating}>
					{creating ? '创建中…' : '创建'}
				</button>
			</div>
		</form>
	</Modal>
{/if}

{#each pages as pageItem (pageItem.id)}
	<div class="card" style="margin-bottom: 16px">
		<div class="card-head">
			<div>
				<h2>{pageItem.name}</h2>
				<div class="small muted">
					<a href={`/status/${pageItem.slug}/`} target="_blank">/status/{pageItem.slug}/</a>
					· {pageItem.has_password ? '有密码' : '公开'}
				</div>
			</div>
			<div class="row">
				<button class="btn btn-sm" type="button" onclick={() => edit(pageItem)}>{editing === pageItem.id ? '收起' : '编辑组件'}</button>
				{#if manageable}<button class="btn btn-sm btn-danger" type="button" onclick={() => remove(pageItem)}>删除</button>{/if}
			</div>
		</div>

		{#if editing === pageItem.id}
			<div class="stack" style="gap: 8px">
				{#each monitors as monitor (monitor.id)}
					<div class="row">
						<label class="row" style="gap: 6px">
							<input
								type="checkbox"
								checked={selected.includes(monitor.id)}
								onchange={(event) => {
									if (event.currentTarget.checked) selected = [...selected, monitor.id];
									else selected = selected.filter((id) => id !== monitor.id);
								}}
							/>
							<span>{monitor.name}</span>
						</label>
						<input
							class="input"
							style="max-width: 220px"
							placeholder="分组名"
							value={groups[monitor.id] ?? ''}
							oninput={(event) => (groups = { ...groups, [monitor.id]: event.currentTarget.value })}
						/>
					</div>
				{/each}
				<div class="row"><button class="btn btn-primary btn-sm" type="button" onclick={() => saveItems(pageItem)}>保存组件</button></div>
			</div>
		{/if}
	</div>
{/each}

{#if pages.length === 0}
	<div class="card"><p class="empty">还没有状态页。</p></div>
{/if}

<p class="small muted">公开地址基于当前站点域名：{origin()}/status/&lt;slug&gt;/</p>
