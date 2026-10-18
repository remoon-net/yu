<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api/client.js';
	import type { Workspace } from '#lib/api/types.js';
	import Modal from '#lib/components/Modal.svelte';
	import { session } from '#lib/stores/session.svelte.js';
	import { roleLabel } from '#lib/utils/roles.js';

	let { children } = $props();

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.workspaces.find((item) => item.slug === slug));

	let showCreate = $state(false);
	let workspaceName = $state('');
	let createError = $state('');
	let creating = $state(false);

	async function createWorkspace(event: SubmitEvent) {
		event.preventDefault();
		const name = workspaceName.trim();
		if (!name) return;

		creating = true;
		createError = '';

		try {
			const created = await api.post<Workspace>('/api/workspaces', { name });
			await session.load(true);
			showCreate = false;
			workspaceName = '';
			await goto(`/app/w/${created.slug}/analytics/`);
		} catch (err) {
			createError = err instanceof Error ? err.message : '创建工作区失败';
		} finally {
			creating = false;
		}
	}

	$effect(() => {
		if (session.loaded && !workspace) {
			void goto('/app/', { replace: true });
		}
	});

	function switchWorkspace(event: Event) {
		const target = event.currentTarget as HTMLSelectElement;
		const next = session.workspaces.find((item) => item.id === target.value);
		if (next) void goto(`/app/w/${next.slug}/analytics/`);
	}

	function toggleTheme() {
		const root = document.documentElement;
		const next = root.dataset.theme === 'light' ? 'dark' : 'light';
		root.dataset.theme = next;
		try {
			localStorage.setItem('yu_theme', next);
		} catch {
			// 忽略隐私模式下的存储失败
		}
	}

	async function logout() {
		await session.logout();
		void goto('/login/', { replace: true });
	}
</script>

{#if workspace}
	<div class="app-shell">
		<aside class="sidebar">
			<div class="brand">
				<span class="brand-mark">Y</span>
				<span>Yu</span>
			</div>

			<div class="workspace-switch">
				<select class="select" value={workspace.id} onchange={switchWorkspace}>
					{#each session.workspaces as item (item.id)}
						<option value={item.id}>{item.name}</option>
					{/each}
				</select>
				<button
					class="btn btn-sm workspace-add"
					type="button"
					title="新建工作区"
					onclick={() => {
						createError = '';
						showCreate = true;
					}}
				>
					＋
				</button>
			</div>

			<nav class="nav">
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/analytics')}
					href={`/app/w/${workspace.slug}/analytics/`}>网站统计</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/monitors')}
					href={`/app/w/${workspace.slug}/monitors/`}>服务监控</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/alerts')}
					href={`/app/w/${workspace.slug}/alerts/`}>告警与事件</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/status-pages')}
					href={`/app/w/${workspace.slug}/status-pages/`}>状态页</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/sites')}
					href={`/app/w/${workspace.slug}/sites/`}>站点设置</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/settings')}
					href={`/app/w/${workspace.slug}/settings/`}>工作区设置</a
				>
			</nav>

			<div class="spacer"></div>

			<div class="small muted" style="padding: 0 6px">
				{session.user?.name} · {session.user?.email}
			</div>
			<div class="row" style="padding: 0 2px">
				<button class="btn btn-sm" type="button" onclick={toggleTheme}>切换主题</button>
				<button class="btn btn-sm" type="button" onclick={logout}>退出</button>
			</div>
		</aside>

		<div class="main">
			<header class="topbar">
				<strong>{workspace.name}</strong>
				<span class="badge">{roleLabel(workspace.role)}</span>
			</header>
			<main class="content">
				{@render children()}
			</main>
		</div>
	</div>

	<Modal open={showCreate} title="新建工作区" size="sm" onclose={() => (showCreate = false)}>
		<form onsubmit={createWorkspace}>
			<label class="field">
				工作区名称
				<input class="input" bind:value={workspaceName} required maxlength={60} placeholder="例如：我的产品" />
			</label>
			{#if createError}<div class="error" style="margin-top: 12px">{createError}</div>{/if}
			<div class="modal-actions">
				<button class="btn" type="button" onclick={() => (showCreate = false)}>取消</button>
				<button class="btn btn-primary" type="submit" disabled={creating}>
					{creating ? '创建中…' : '创建'}
				</button>
			</div>
		</form>
	</Modal>
{:else}
	<div class="boot">加载中…</div>
{/if}

<style>
	.workspace-switch {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.workspace-switch .select {
		flex: 1;
		min-width: 0;
	}

	.workspace-add {
		height: 38px;
		width: 38px;
		flex: 0 0 auto;
		padding: 0;
		font-size: 16px;
		line-height: 1;
	}
</style>
