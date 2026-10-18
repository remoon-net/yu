<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import type { Site } from '#lib/api/types.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let sites = $state<Site[]>([]);
	let loading = $state(true);
	let error = $state('');
	let showCreate = $state(false);
	let creating = $state(false);

	let name = $state('');
	let domain = $state('');
	let timezone = $state('Asia/Shanghai');

	const timezones = [
		'Asia/Shanghai',
		'Asia/Tokyo',
		'Asia/Singapore',
		'UTC',
		'Europe/London',
		'Europe/Berlin',
		'America/New_York',
		'America/Los_Angeles'
	];

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

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace) return;

		creating = true;
		error = '';

		try {
			await api.post<Site>(`/api/workspaces/${workspace.id}/sites`, {
				name,
				domain,
				timezone
			});

			name = '';
			domain = '';
			showCreate = false;

			sites = await api.get<Site[]>(`/api/workspaces/${workspace.id}/sites`);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建站点失败';
		} finally {
			creating = false;
		}
	}
</script>

<div class="page-head">
	<div>
		<h1>网站统计</h1>
		<p class="small muted">选择站点查看看板，或创建新的站点。</p>
	</div>

	{#if manageable}
		<button class="btn btn-primary" type="button" onclick={() => (showCreate = !showCreate)}>
			{showCreate ? '取消' : '添加站点'}
		</button>
	{/if}
</div>

{#if error}
	<div class="error" style="margin-bottom: 16px">{error}</div>
{/if}

{#if showCreate}
	<form class="card" onsubmit={create} style="margin-bottom: 16px">
		<div class="grid-2">
			<label class="field">
				站点名称
				<input class="input" bind:value={name} required maxlength={60} placeholder="例如：官网" />
			</label>

			<label class="field">
				域名（用于识别站内来源）
				<input class="input" bind:value={domain} placeholder="example.com" />
			</label>

			<label class="field">
				时区
				<select class="select" bind:value={timezone}>
					{#each timezones as zone (zone)}
						<option value={zone}>{zone}</option>
					{/each}
				</select>
			</label>
		</div>

		<div class="row" style="margin-top: 14px">
			<button class="btn btn-primary" type="submit" disabled={creating}>
				{creating ? '创建中…' : '创建站点'}
			</button>
		</div>
	</form>
{/if}

{#if loading}
	<p class="empty">加载中…</p>
{:else if sites.length === 0}
	<div class="card">
		<p class="empty">还没有站点。添加一个站点并嵌入采集脚本即可开始统计。</p>
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
					<a class="btn btn-sm btn-primary" href={`/app/w/${slug}/analytics/${site.id}/`}>查看看板</a>
					<a class="btn btn-sm" href={`/app/w/${slug}/sites/${site.id}/`}>安装与设置</a>
				</div>
			</div>
		{/each}
	</div>
{/if}
