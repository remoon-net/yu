<script lang="ts">
	import { page } from '$app/state';
	import { api } from '#lib/api/client.js';
	import type { Site } from '#lib/api/types.js';
	import { session } from '#lib/stores/session.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));

	let sites = $state<Site[]>([]);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		if (!workspace) return;

		const workspaceId = workspace.id;
		let cancelled = false;

		loading = true;
		error = '';

		api
			.get<Site[]>(`/api/workspaces/${workspaceId}/sites`)
			.then((result) => {
				if (!cancelled) sites = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载站点失败';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});
</script>

<div class="page-head">
	<div>
		<h1>站点设置</h1>
		<p class="small muted">管理采集脚本、域名与时区。</p>
	</div>
	<a class="btn btn-sm" href={`/app/w/${slug}/analytics`}>前往统计</a>
</div>

{#if error}
	<div class="error">{error}</div>
{/if}

{#if loading}
	<p class="empty">加载中…</p>
{:else if sites.length === 0}
	<div class="card">
		<p class="empty">还没有站点，请先在「网站统计」中添加。</p>
	</div>
{:else}
	<div class="grid-2">
		{#each sites as site (site.id)}
			<div class="card site-card">
				<div>
					<div class="site-name">{site.name}</div>
					<div class="small muted">{site.domain || '未设置域名'} · {site.timezone}</div>
				</div>

				<div class="row">
					<a class="btn btn-sm" href={`/app/w/${slug}/sites/${site.id}`}>打开设置</a>
					<a class="btn btn-sm" href={`/app/w/${slug}/analytics/${site.id}`}>查看统计</a>
				</div>
			</div>
		{/each}
	</div>
{/if}
