<script lang="ts">
	import { page } from '$app/state';
	import { api } from '#lib/api/client.js';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { Site, StatsFilter } from '#lib/api/types.js';
	import AudiencePanel from '#lib/components/analytics/AudiencePanel.svelte';
	import EventsPanel from '#lib/components/analytics/EventsPanel.svelte';
	import FunnelPanel from '#lib/components/analytics/FunnelPanel.svelte';
	import GoalsPanel from '#lib/components/analytics/GoalsPanel.svelte';
	import OverviewPanel from '#lib/components/analytics/OverviewPanel.svelte';
	import PagesPanel from '#lib/components/analytics/PagesPanel.svelte';
	import RealtimePanel from '#lib/components/analytics/RealtimePanel.svelte';
	import RetentionPanel from '#lib/components/analytics/RetentionPanel.svelte';
	import SourcesPanel from '#lib/components/analytics/SourcesPanel.svelte';
	import StatsFilterBar from '#lib/components/analytics/StatsFilterBar.svelte';
	import RangePicker from '#lib/components/RangePicker.svelte';
	import { canManage, session } from '#lib/stores/session.svelte.js';

	type Tab =
		| 'overview'
		| 'realtime'
		| 'pages'
		| 'sources'
		| 'audience'
		| 'events'
		| 'goals'
		| 'funnel'
		| 'retention';

	const tabs: { key: Tab; label: string }[] = [
		{ key: 'overview', label: '概览' },
		{ key: 'realtime', label: '实时' },
		{ key: 'retention', label: '留存' },
		{ key: 'pages', label: '页面' },
		{ key: 'sources', label: '来源' },
		{ key: 'audience', label: '访客画像' },
		{ key: 'events', label: '事件' },
		{ key: 'goals', label: '目标' },
		{ key: 'funnel', label: '漏斗' }
	];

	const slug = $derived(page.params.ws ?? '');
	const siteId = $derived(page.params.siteId ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let site = $state<Site | null>(null);
	let error = $state('');
	let range = $state('24h');
	let tab = $state<Tab>('overview');
	let filters = $state<StatsFilter>({});

	$effect(() => {
		if (!workspace || !siteId) return;

		const workspaceId = workspace.id;
		let cancelled = false;

		site = null;
		error = '';
		filters = {};

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
			<a class="btn btn-sm" href={analyticsApi.exportUrl(workspace.id, site.id, range)}>导出 CSV</a>
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

	{#if tab !== 'realtime'}
		<StatsFilterBar
			workspaceId={workspace.id}
			siteId={site.id}
			{filters}
			onApply={(next) => (filters = next)}
		/>
	{/if}

	{#if tab === 'overview'}
		<OverviewPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			timezone={site.timezone}
			filter={filters}
		/>
	{:else if tab === 'realtime'}
		<RealtimePanel workspaceId={workspace.id} siteId={site.id} />
	{:else if tab === 'retention'}
		<RetentionPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			filter={filters}
		/>
	{:else if tab === 'pages'}
		<PagesPanel workspaceId={workspace.id} siteId={site.id} {range} filter={filters} />
	{:else if tab === 'sources'}
		<SourcesPanel workspaceId={workspace.id} siteId={site.id} {range} filter={filters} />
	{:else if tab === 'audience'}
		<AudiencePanel workspaceId={workspace.id} siteId={site.id} {range} filter={filters} />
	{:else if tab === 'events'}
		<EventsPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			timezone={site.timezone}
			filter={filters}
		/>
	{:else if tab === 'goals'}
		<GoalsPanel
			workspaceId={workspace.id}
			siteId={site.id}
			{range}
			{manageable}
			filter={filters}
		/>
	{:else if tab === 'funnel'}
		<FunnelPanel workspaceId={workspace.id} siteId={site.id} {range} filter={filters} />
	{/if}
{:else if !error}
	<p class="empty">加载中…</p>
{/if}
