<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api/client.js';
	import type { Workspace } from '#lib/api/types.js';
	import Icon from '#lib/components/Icon.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import { session } from '#lib/stores/session.svelte.js';
	import { roleLabel } from '#lib/utils/roles.js';

	let { children } = $props();

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.workspaces.find((item) => item.slug === slug));

	const navItems = [
		{ key: 'analytics', label: '网站统计', icon: 'chart' },
		{ key: 'monitors', label: '服务监控', icon: 'pulse' },
		{ key: 'alerts', label: '告警与事件', icon: 'bell' },
		{ key: 'status-pages', label: '状态页', icon: 'activity' },
		{ key: 'short-links', label: '短链', icon: 'link' },
		{ key: 'pixels', label: '遥测', icon: 'eye' },
		{ key: 'sites', label: '站点设置', icon: 'globe' },
		{ key: 'settings', label: '工作区设置', icon: 'settings' }
	];

	const nav = $derived(
		navItems.map((item) => ({
			...item,
			href: `/app/w/${slug}/${item.key}/`
		}))
	);

	const pathname = $derived(decodeURIComponent(page.url.pathname));
	const current = $derived(
		[...nav].sort((a, b) => b.href.length - a.href.length).find((item) =>
			pathname.startsWith(item.href)
		)
	);

	const userInitial = $derived(
		(session.user?.name || session.user?.email || 'U').slice(0, 1).toUpperCase()
	);

	let dark = $state(
		typeof document !== 'undefined' && document.documentElement.dataset.theme === 'dark'
	);

	let showCreate = $state(false);
	let workspaceName = $state('');
	let workspaceSlug = $state('');
	let createError = $state('');
	let creating = $state(false);
	let loggingOut = $state(false);

	async function createWorkspace(event: SubmitEvent) {
		event.preventDefault();
		const name = workspaceName.trim();
		if (!name) return;

		creating = true;
		createError = '';

		try {
			const created = await api.post<Workspace>('/api/workspaces', {
				name,
				slug: workspaceSlug.trim().toLowerCase()
			});
			await session.load(true);
			showCreate = false;
			workspaceName = '';
			workspaceSlug = '';
			await goto(`/app/w/${created.slug}/analytics/`);
		} catch (err) {
			createError = err instanceof Error ? err.message : '创建工作区失败';
		} finally {
			creating = false;
		}
	}

	// 工作区可能已在别处被删除：先刷新一次会话再判断，避免停留在失效路径上。
	let reloadingWorkspace = false;

	$effect(() => {
		if (!session.loaded || workspace) return;
		if (reloadingWorkspace) return;

		reloadingWorkspace = true;
		void session.load(true).finally(() => {
			reloadingWorkspace = false;
			if (!session.findWorkspace(slug)) void goto('/app/', { replace: true });
		});
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
		dark = next === 'dark';
		try {
			localStorage.setItem('yu_theme', next);
		} catch {
			// 忽略隐私模式下的存储失败
		}
	}

	async function logout() {
		loggingOut = true;
		try {
			await session.logout();
			void goto('/login/', { replace: true });
		} finally {
			loggingOut = false;
		}
	}
</script>

{#if workspace}
	<div class="app-shell">
		<aside class="sidebar">
			<a class="brand" href={`/app/w/${workspace.slug}/analytics/`}>
				<span class="brand-mark">Y</span>
				<span>Yu</span>
			</a>

			<div class="workspace-switch">
				<span class="switch-label">工作区</span>
				<div class="workspace-row">
					<select
						class="select"
						aria-label="切换工作区"
						value={workspace.id}
						onchange={switchWorkspace}
					>
						{#each session.workspaces as item (item.id)}
							<option value={item.id}>{item.name}</option>
						{/each}
					</select>
					<button
						class="icon-btn workspace-add"
						type="button"
						title="新建工作区"
						aria-label="新建工作区"
						onclick={() => {
							createError = '';
							showCreate = true;
						}}
					>
						<Icon name="plus" size={15} />
					</button>
				</div>
			</div>

			<nav class="nav" aria-label="主导航">
				{#each nav as item (item.key)}
					<a
						class="nav-item"
						class:active={current?.key === item.key}
						aria-current={current?.key === item.key ? 'page' : undefined}
						href={item.href}
					>
						<Icon name={item.icon} size={16} />
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>

			<div class="sidebar-foot">
				<div class="user-card">
					<span class="avatar">{userInitial}</span>
					<div class="who">
						<span class="truncate">{session.user?.name}</span>
						<span class="tiny dim truncate">{session.user?.email}</span>
					</div>
				</div>
			</div>
		</aside>

		<div class="main">
			<header class="topbar">
				<div class="topbar-title">{current?.label ?? 'Yu'}</div>
				<div class="topbar-actions">
					<span class="badge role-tag">{roleLabel(workspace.role)}</span>
					<button
						class="icon-btn"
						type="button"
						onclick={toggleTheme}
						title={dark ? '切换到亮色' : '切换到暗色'}
						aria-label={dark ? '切换到亮色' : '切换到暗色'}
					>
						<Icon name={dark ? 'sun' : 'moon'} size={16} />
					</button>
					<details class="user-menu">
						<summary aria-label="用户菜单">
							<span class="avatar user-avatar">{userInitial}</span>
						</summary>
						<div class="menu">
							<div class="menu-head">
								<span class="menu-name">{session.user?.name}</span>
								<span class="tiny dim truncate">{session.user?.email}</span>
							</div>
							<div class="divider"></div>
							<button class="menu-item danger" type="button" disabled={loggingOut} onclick={logout}>
								<Icon name="logout" size={14} />
								{loggingOut ? '退出中…' : '退出登录'}
							</button>
						</div>
					</details>
				</div>
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
			<label class="field" style="margin-top: 12px">
				工作区标识（可选）
				<input
					class="input"
					bind:value={workspaceSlug}
					pattern={"[a-z0-9][a-z0-9\\-]{0,38}[a-z0-9]"}
					maxlength={40}
					placeholder="my-team"
				/>
			</label>
			<p class="small muted" style="margin: 8px 0 0">
				路径会变成 /app/w/&lt;标识&gt;/；小写字母、数字与连字符，留空自动生成 ws-xxxxxx。
			</p>
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
		flex-direction: column;
		gap: 5px;
		padding: 0 4px;
	}

	.switch-label {
		font-size: 11px;
		line-height: 1;
		letter-spacing: 0.05em;
		color: var(--muted);
		padding-left: 2px;
	}

	.workspace-row {
		display: flex;
		gap: 6px;
		align-items: center;
	}

	.workspace-row .select {
		flex: 1;
		min-width: 0;
		height: 34px;
	}

	.workspace-add {
		width: 34px;
		height: 34px;
		flex: 0 0 auto;
		border: 1px solid var(--border);
		background: var(--panel);
	}

	.workspace-add:hover {
		border-color: var(--border-strong);
		background: var(--panel-2);
	}

	.user-menu {
		position: relative;
	}

	.user-menu summary {
		display: block;
		list-style: none;
		cursor: pointer;
	}

	.user-menu summary::-webkit-details-marker {
		display: none;
	}

	.user-avatar {
		width: 30px;
		height: 30px;
		border-radius: 50%;
		background: var(--accent-soft);
		color: var(--accent);
		transition: box-shadow 0.15s ease;
	}

	.user-menu summary:hover .user-avatar {
		box-shadow: 0 0 0 3px var(--accent-soft);
	}

	.menu {
		position: absolute;
		right: 0;
		top: calc(100% + 8px);
		z-index: 60;
		width: 210px;
		padding: 6px;
		background: var(--panel);
		border: 1px solid var(--border);
		border-radius: 12px;
		box-shadow: var(--shadow-pop);
	}

	.menu-head {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 8px 9px;
		font-size: 12.5px;
	}

	.menu-name {
		font-weight: 600;
	}

	.menu-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 8px 9px;
		border: 0;
		border-radius: 8px;
		background: transparent;
		color: var(--text);
		font: inherit;
		font-size: 12.5px;
		text-align: left;
		cursor: pointer;
	}

	.menu-item:hover {
		background: var(--panel-2);
	}

	.menu-item.danger {
		color: var(--danger);
	}

	@media (max-width: 860px) {
		.workspace-switch {
			flex: 1;
			flex-direction: row;
			align-items: center;
			padding: 0;
		}

		.workspace-row {
			flex: 1;
		}

		.switch-label,
		.workspace-add {
			display: none;
		}
	}
</style>
