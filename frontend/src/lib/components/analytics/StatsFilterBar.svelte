<script lang="ts">
	import { ApiError } from '#lib/api/client.js';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { Segment, StatsFilter } from '#lib/api/types.js';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';

	let {
		workspaceId,
		siteId,
		filters,
		onApply
	}: {
		workspaceId: string;
		siteId: string;
		filters: StatsFilter;
		onApply: (next: StatsFilter) => void;
	} = $props();

	let draft = $state<StatsFilter>({});
	let segments = $state<Segment[]>([]);
	let selectedId = $state('');
	let segmentName = $state('');
	let saving = $state(false);
	let deleting = $state(false);
	let error = $state('');

	$effect(() => {
		draft = { ...filters };
	});

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		let cancelled = false;

		segments = [];
		selectedId = '';

		analyticsApi
			.segments(ws, site)
			.then((result) => {
				if (!cancelled) segments = result.segments;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载分段失败';
			});

		return () => {
			cancelled = true;
		};
	});

	const activeCount = $derived(
		Object.values(filters).filter((value) => (value ?? '').trim().length > 0).length
	);
	const draftCount = $derived(
		Object.values(draft).filter((value) => (value ?? '').trim().length > 0).length
	);

	function apply(event?: SubmitEvent) {
		event?.preventDefault();
		onApply({ ...draft });
	}

	function reset() {
		draft = {};
		selectedId = '';
		onApply({});
	}

	function chooseSegment() {
		const segment = segments.find((item) => item.id === selectedId);
		if (!segment) return;

		draft = { ...segment.filters };
		onApply({ ...segment.filters });
	}

	async function reloadSegments(selectId = '') {
		const result = await analyticsApi.segments(workspaceId, siteId);
		segments = result.segments;
		selectedId = selectId;
	}

	async function saveSegment() {
		const name = segmentName.trim();
		if (!name || draftCount === 0) return;

		saving = true;
		error = '';
		try {
			const segment = await analyticsApi.createSegment(workspaceId, siteId, {
				name,
				filters: { ...draft }
			});
			segmentName = '';
			onApply({ ...draft });
			await reloadSegments(segment.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '保存分段失败';
		} finally {
			saving = false;
		}
	}

	async function removeSegment() {
		const segment = segments.find((item) => item.id === selectedId);
		if (!segment) return;

		const ok = await confirmDialog({
			title: '删除分段',
			message: `确定删除分段「${segment.name}」？`,
			confirmText: '删除'
		});
		if (!ok) return;

		deleting = true;
		error = '';
		try {
			await analyticsApi.deleteSegment(workspaceId, siteId, segment.id);
			await reloadSegments();
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除分段失败';
		} finally {
			deleting = false;
		}
	}
</script>

<div class="card" style="margin-bottom: 16px">
	<div class="card-head">
		<h2>筛选</h2>
		<span class="small muted">
			{activeCount > 0 ? `已应用 ${activeCount} 个条件` : '按访客筛选当前站点的统计'}
		</span>
	</div>

	{#if error}
		<div class="error" style="margin-bottom: 12px">{error}</div>
	{/if}

	<form onsubmit={apply}>
		<div
			style="display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:10px"
		>
			<label class="field">页面路径
				<input class="input" bind:value={draft.path} placeholder="/pricing" maxlength={256} />
			</label>
			<label class="field">来源域名
				<input
					class="input"
					bind:value={draft.referrer_host}
					placeholder="google.com"
					maxlength={256}
				/>
			</label>
			<label class="field">国家
				<input class="input" bind:value={draft.country} placeholder="CN" maxlength={256} />
			</label>
			<label class="field">设备
				<input class="input" bind:value={draft.device} placeholder="Desktop / Mobile" maxlength={256} />
			</label>
			<label class="field">浏览器
				<input class="input" bind:value={draft.browser} placeholder="Chrome" maxlength={256} />
			</label>
			<label class="field">UTM Source
				<input
					class="input"
					bind:value={draft.utm_source}
					placeholder="newsletter"
					maxlength={256}
				/>
			</label>
		</div>

		<div class="row" style="margin-top: 12px">
			<button class="btn btn-sm btn-primary" type="submit">应用筛选</button>
			<button class="btn btn-sm" type="button" onclick={reset} disabled={activeCount === 0 && draftCount === 0}>
				重置
			</button>
		</div>
	</form>

	<div class="row" style="margin-top: 14px; align-items: end; gap: 10px; flex-wrap: wrap">
		<label class="field" style="min-width: 180px">
			已保存分段
			<select class="select" bind:value={selectedId} onchange={chooseSegment}>
				<option value="">选择分段</option>
				{#each segments as segment (segment.id)}
					<option value={segment.id}>{segment.name}</option>
				{/each}
			</select>
		</label>

		<label class="field" style="min-width: 180px">
			保存当前筛选
			<input
				class="input"
				bind:value={segmentName}
				placeholder="分段名称"
				maxlength={80}
			/>
		</label>

		<button
			class="btn btn-sm"
			type="button"
			onclick={saveSegment}
			disabled={saving || !segmentName.trim() || draftCount === 0}
		>
			{saving ? '保存中…' : '保存为分段'}
		</button>

		{#if selectedId}
			<button class="btn btn-sm" type="button" onclick={removeSegment} disabled={deleting}>
				{deleting ? '删除中…' : '删除分段'}
			</button>
		{/if}
	</div>
</div>
