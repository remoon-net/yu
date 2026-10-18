<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { session } from '#lib/stores/session.svelte.js';
	import { roleLabel } from '#lib/utils/roles.js';

	let { children } = $props();

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.workspaces.find((item) => item.slug === slug));

	$effect(() => {
		if (session.loaded && !workspace) {
			void goto('/app', { replace: true });
		}
	});

	function switchWorkspace(event: Event) {
		const target = event.currentTarget as HTMLSelectElement;
		const next = session.workspaces.find((item) => item.id === target.value);
		if (next) void goto(`/app/w/${next.slug}/analytics`);
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
		void goto('/login', { replace: true });
	}
</script>

{#if workspace}
	<div class="app-shell">
		<aside class="sidebar">
			<div class="brand">
				<span class="brand-mark">Y</span>
				<span>Yu</span>
			</div>

			<select class="select" value={workspace.id} onchange={switchWorkspace}>
				{#each session.workspaces as item (item.id)}
					<option value={item.id}>{item.name}</option>
				{/each}
			</select>

			<nav class="nav">
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/analytics')}
					href={`/app/w/${workspace.slug}/analytics`}>网站统计</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/sites')}
					href={`/app/w/${workspace.slug}/sites`}>站点设置</a
				>
				<a
					class="nav-item"
					class:active={page.url.pathname.includes('/settings')}
					href={`/app/w/${workspace.slug}/settings`}>工作区设置</a
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
{:else}
	<div class="boot">加载中…</div>
{/if}
