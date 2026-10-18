<script lang="ts">
	import { page } from '$app/state';
	import { api } from '#lib/api/client.js';
	import type { Site } from '#lib/api/types.js';
	import AudiencePanel from '#lib/components/analytics/AudiencePanel.svelte';
	import EventsPanel from '#lib/components/analytics/EventsPanel.svelte';
	import OverviewPanel from '#lib/components/analytics/OverviewPanel.svelte';
	import PagesPanel from '#lib/components/analytics/PagesPanel.svelte';
	import RealtimePanel from '#lib/components/analytics/RealtimePanel.svelte';
	import SourcesPanel from '#lib/components/analytics/SourcesPanel.svelte';
	import RangePicker from '#lib/components/RangePicker.svelte';
	import { session } from '#lib/stores/session.svelte.js';

	type Tab = 'overview' | 'realtime' | 'pages' | 'sources' | 'audience' | 'events';

	const tabs: { key: Tab; label: string }[] = [
		{ key: 'overview', label: '概览' },
		{ key: 'realtime', label: '实时' },
		{ key: 'pages', label: '页面' },
		{ key: 'sources', label: '来源' },
		{ key: 'audience', label: '访客画像' },
		{ key: 'events', label: '事件' }
	];

	const slug = $derived(page.params.ws ?? '');
	const siteId = $derived(page.params.siteId ?? '');
	const workspace = $derived(session.findWorkspace(slug));

	let site = $state<Site | null>(null);
	let error = $state('');
	let range = $state('24h');
	let tab = $state<Tab>('overview');

	$effect(() => {
		if (!workspace || !siteId) return;

		const workspaceId = workspace.id;
		let cancelled = false;

		site = null;
		error = '';

		api
			.get<Site>(`/api/workspaces/${workspaceId}/sites/${siteId}`)
			.then((result) => {
				if (!cancelled) site = result;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载站点失败';
			});

		return () => {
			cancelled = true;
		};
	});
</script>

{#if error}
	<div class="error">{error}</div>
{/if}

{#if site && workspace}
	<div class="page-head">
		<div>
			<h1>{site.name}</h1>
			<p class="small muted">{site.domain || '未设置域名'} · {site.timezone}</p>
		</div>

		<div class="row">
			{#if tab !== 'realtime'}
				<RangePicker bind:value={range} />
			{/if}
			<a class="btn btn-sm" href={`/app/w/${slug}/sites/${site.id}/`}>设置</a>
		</div>
	</div>

	<div class="tabs">
		{#each tabs as item (item.key)}
			<button
				type="button"
				class="tab"
				class:active={tab === item.key}
				onclick={() => (tab = item.key)}>{item.label}</button
			>
		{/each}
	</div>

	{#if tab === 'overview'}
		<OverviewPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			timezone={site.timezone}
		/>
	{:else if tab === 'realtime'}
		<RealtimePanel workspaceId={workspace.id} siteId={site.id} />
	{:else if tab === 'pages'}
		<PagesPanel workspaceId={workspace.id} siteId={site.id} {range} />
	{:else if tab === 'sources'}
		<SourcesPanel workspaceId={workspace.id} siteId={site.id} {range} />
	{:else if tab === 'audience'}
		<AudiencePanel workspaceId={workspace.id} siteId={site.id} {range} />
	{:else if tab === 'events'}
		<EventsPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			timezone={site.timezone}
		/>
	{/if}
{:else if !error}
	<p class="empty">加载中…</p>
{/if}
