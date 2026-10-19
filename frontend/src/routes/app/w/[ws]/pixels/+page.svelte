<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import { pixelApi } from '#lib/api/monitors.js';
	import type { Pixel, PixelStats, PixelStatsResponse, Site } from '#lib/api/types.js';
	import HitDetail from '#lib/components/HitDetail.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';
	import { formatDateTime, formatNumber } from '#lib/utils/format.js';
	import { createPending } from '#lib/utils/pending.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));
	const pending = createPending();

	let sites = $state<Site[]>([]);
	let siteId = $state('');
	let pixels = $state<Pixel[]>([]);
	let stats = $state<Record<string, PixelStats>>({});
	let sitesLoading = $state(true);
	let pixelsLoading = $state(true);
	let error = $state('');
	let notice = $state('');

	let showForm = $state(false);
	let editing = $state<Pixel | null>(null);
	let formSlug = $state('');
	let formName = $state('');
	let formEnabled = $state(true);
	let formExpires = $state('');
	let formError = $state('');
	let saving = $state(false);

	let detailTarget = $state<Pixel | null>(null);
	let detail = $state<PixelStatsResponse | null>(null);
	let detailRange = $state('30d');
	let detailLoading = $state(false);
	let detailError = $state('');

	const currentSite = $derived(sites.find((site) => site.id === siteId));
	const timezone = $derived(currentSite?.timezone || 'UTC');

	$effect(() => {
		if (!workspace) return;
		sitesLoading = true;
		siteId = '';
		pixels = [];
		stats = {};
		void loadSites(workspace.id);
	});

	$effect(() => {
		if (!workspace || !siteId) return;
		pixelsLoading = true;
		void loadPixels(workspace.id, siteId);
	});

	async function loadSites(workspaceId: string) {
		error = '';
		try {
			sites = await api.get<Site[]>(`/api/workspaces/${workspaceId}/sites`);
			if (!siteId && sites.length > 0) siteId = sites[0].id;
		} catch (err) {
			error = err instanceof Error ? err.message : '加载站点失败';
		} finally {
			sitesLoading = false;
		}
	}

	async function loadPixels(workspaceId: string, currentSiteId: string) {
		try {
			const data = await pixelApi.list(workspaceId, currentSiteId);
			pixels = data.pixels ?? [];
			stats = Object.fromEntries((data.stats ?? []).map((item) => [item.slug, item]));
		} catch (err) {
			error = err instanceof Error ? err.message : '加载像素失败';
		} finally {
			pixelsLoading = false;
		}
	}

	function pixelURL(pixelSlug: string) {
		return `${window.location.origin}/t/${pixelSlug}.gif`;
	}

	function snippet(pixel: Pixel) {
		return `<img src="${pixelURL(pixel.slug)}" width="1" height="1" alt="" />`;
	}

	function statOf(pixel: Pixel): PixelStats | undefined {
		return stats[pixel.slug];
	}

	function statusOf(pixel: Pixel) {
		if (!pixel.enabled) return '已停用';
		if (pixel.expires_at && new Date(pixel.expires_at) < new Date()) return '已过期';
		return '启用中';
	}

	function toLocalInput(value: string | null) {
		if (!value) return '';
		const date = new Date(value);
		const pad = (n: number) => String(n).padStart(2, '0');
		return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
	}

	function openCreate() {
		editing = null;
		formSlug = '';
		formName = '';
		formEnabled = true;
		formExpires = '';
		formError = '';
		showForm = true;
	}

	function openEdit(pixel: Pixel) {
		editing = pixel;
		formSlug = pixel.slug;
		formName = pixel.name;
		formEnabled = pixel.enabled;
		formExpires = toLocalInput(pixel.expires_at);
		formError = '';
		showForm = true;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!workspace || !siteId) return;

		saving = true;
		formError = '';

		const body = {
			slug: formSlug.trim().toLowerCase(),
			name: formName.trim(),
			enabled: formEnabled,
			expires_at: formExpires ? new Date(formExpires).toISOString() : null
		};

		try {
			if (editing) {
				await pixelApi.update(workspace.id, siteId, editing.id, body);
				notice = '像素已更新';
			} else {
				await pixelApi.create(workspace.id, siteId, body);
				notice = '像素已创建';
			}
			showForm = false;
			editing = null;
			await loadPixels(workspace.id, siteId);
		} catch (err) {
			formError = err instanceof ApiError ? err.message : '保存像素失败';
		} finally {
			saving = false;
		}
	}

	async function remove(pixel: Pixel) {
		if (!workspace || !siteId) return;

		// 从编辑弹窗触发时，先关闭编辑弹窗再弹确认，取消可返回。
		const fromForm = editing?.id === pixel.id;
		if (fromForm) showForm = false;

		const ok = await confirmDialog({
			title: '删除像素',
			message: `确定删除像素「${pixel.slug}」？删除后已嵌入的图片将不再记录曝光。`,
			confirmText: '删除'
		});

		if (!ok) {
			if (fromForm) showForm = true;
			return;
		}

		try {
			await pending.run(`delete:${pixel.id}`, async () => {
				await pixelApi.remove(workspace.id, siteId, pixel.id);
				editing = null;
				notice = '像素已删除';
				await loadPixels(workspace.id, siteId);
			});
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除像素失败';
		}
	}

	async function copySnippet(pixel: Pixel) {
		try {
			await navigator.clipboard.writeText(snippet(pixel));
			notice = '嵌入代码已复制';
		} catch {
			error = '复制失败，请手动选择代码';
		}
	}

	async function openDetail(pixel: Pixel) {
		await pending.run(`detail:${pixel.id}`, async () => {
			detailTarget = pixel;
			detail = null;
			detailError = '';
			await loadDetail();
		});
	}

	async function loadDetail() {
		if (!workspace || !siteId || !detailTarget) return;
		detailLoading = true;
		detailError = '';
		try {
			detail = await pixelApi.stats(workspace.id, siteId, detailTarget.id, detailRange);
		} catch (err) {
			detailError = err instanceof Error ? err.message : '加载像素统计失败';
		} finally {
			detailLoading = false;
		}
	}

	async function setDetailRange(value: string) {
		if (detailRange === value) return;
		detailRange = value;
		await loadDetail();
	}
</script>

<div class="page-head">
	<div>
		<h1>遥测</h1>
		<p class="small muted">通过 1×1 透明像素上报曝光，适用于无法注入采集脚本的页面。</p>
	</div>
	<div class="row">
		{#if sites.length > 1}
			<select class="select site-select" bind:value={siteId}>
				{#each sites as site (site.id)}<option value={site.id}>{site.name}</option>{/each}
			</select>
		{/if}
		{#if manageable}
			<button class="btn btn-primary" type="button" onclick={openCreate} disabled={sites.length === 0}>新建像素</button>
		{/if}
	</div>
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

{#if sitesLoading}
	<div class="card"><p class="empty">加载中…</p></div>
{:else if sites.length === 0}
	<div class="card"><p class="empty">还没有站点，请先在「站点设置」中添加站点。</p></div>
{:else if pixelsLoading}
	<div class="card"><p class="empty">加载中…</p></div>
{:else if pixels.length === 0}
	<div class="card"><p class="empty">还没有遥测像素。</p></div>
{:else}
	<div class="card table-card">
		<table class="table pixel-table">
			<thead>
				<tr>
					<th class="col-name">像素</th>
					<th class="num col-shrink">曝光</th>
					<th class="num col-shrink">访客</th>
					<th class="col-shrink">状态</th>
					<th class="col-shrink">最近曝光</th>
					<th class="col-shrink"></th>
				</tr>
			</thead>
			<tbody>
				{#each pixels as pixel (pixel.id)}
					{@const item = statOf(pixel)}
					<tr>
						<td>
							<div class="slug-line"><code>/t/{pixel.slug}.gif</code></div>
							{#if pixel.name}<div class="small muted ellipsis" title={pixel.name}>{pixel.name}</div>{/if}
						</td>
						<td class="num metric-cell">{formatNumber(item?.hits ?? 0)}</td>
						<td class="num metric-cell">{formatNumber(item?.visitors ?? 0)}</td>
						<td>
							<span
								class="badge"
								class:ok={statusOf(pixel) === '启用中'}
								class:badge-idle={statusOf(pixel) !== '启用中'}
							>
								{statusOf(pixel)}
							</span>
						</td>
						<td class="small muted">
							{item?.last_hit ? formatDateTime(item.last_hit, timezone) : '—'}
						</td>
						<td class="actions-cell">
							<div class="actions">
								<button class="btn btn-sm" type="button" disabled={pending.has(`detail:${pixel.id}`)} onclick={() => openDetail(pixel)}>{pending.has(`detail:${pixel.id}`) ? '加载中…' : '详情'}</button>
								<button
									class="icon-btn"
									type="button"
									title="复制嵌入代码"
									aria-label="复制嵌入代码"
									onclick={() => copySnippet(pixel)}
								>
									<svg viewBox="0 0 24 24" aria-hidden="true">
										<rect x="9" y="9" width="11" height="11" rx="2" />
										<path d="M5 15V5a2 2 0 0 1 2-2h10" />
									</svg>
								</button>
								{#if manageable}
									<button class="icon-btn" type="button" title="编辑" aria-label="编辑" onclick={() => openEdit(pixel)}>
										<svg viewBox="0 0 24 24" aria-hidden="true">
											<path d="M4 20h4L20 8l-4-4L4 16v4z" />
											<path d="M14 6l4 4" />
										</svg>
									</button>
								{/if}
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<Modal open={showForm} title={editing ? '编辑像素' : '新建像素'} onclose={() => (showForm = false)}>
	<form onsubmit={submit}>
		<div class="grid-2">
			<label class="field">
				名称
				<input class="input" bind:value={formName} maxlength={80} placeholder="例如：掘金文章引流" />
			</label>
			<label class="field">
				像素标识{editing ? '' : '（可选）'}
				{#if editing}
					<div class="readonly-value"><code>/t/{formSlug}.gif</code><span class="small muted">创建后不可修改</span></div>
				{:else}
					<input
						class="input"
						bind:value={formSlug}
						pattern={"[a-z0-9][a-z0-9\\-]{0,38}[a-z0-9]"}
						maxlength={40}
						placeholder="留空自动生成"
					/>
				{/if}
			</label>
			<label class="field">过期时间（可选）<input class="input" type="datetime-local" bind:value={formExpires} /></label>
		</div>
		<label class="row" style="margin-top: 14px; gap: 8px; font-size: 13px">
			<input type="checkbox" bind:checked={formEnabled} /> 启用该像素
		</label>
		{#if formError}<div class="error" style="margin-top: 12px">{formError}</div>{/if}
		<div class="modal-actions">
			{#if editing}
				<button
					class="btn btn-danger"
					type="button"
					style="margin-right: auto"
					disabled={editing ? pending.has(`delete:${editing.id}`) : false}
					onclick={() => remove(editing!)}
				>
					{editing && pending.has(`delete:${editing.id}`) ? '删除中…' : '删除'}
				</button>
			{/if}
			<button class="btn" type="button" onclick={() => (showForm = false)}>取消</button>
			<button class="btn btn-primary" type="submit" disabled={saving}>{saving ? '保存中…' : '保存'}</button>
		</div>
	</form>
</Modal>

<Modal
	open={!!detailTarget}
	title={`像素详情 · /t/${detailTarget?.slug ?? ''}.gif`}
	size="lg"
	onclose={() => (detailTarget = null)}
>
	{#if detailTarget}
		<HitDetail
			stats={detail?.stats ?? null}
			rangeInfo={detail?.range ?? null}
			{timezone}
			range={detailRange}
			loading={detailLoading}
			error={detailError}
			target={{ text: pixelURL(detailTarget.slug) }}
			targetLabel="像素地址"
			countLabel="曝光"
			onRangeChange={setDetailRange}
		/>
	{/if}
</Modal>

<style>
	.site-select {
		width: auto;
		min-width: 150px;
	}

	.table-card {
		padding: 4px 0;
		overflow-x: auto;
	}

	.pixel-table {
		width: 100%;
	}

	.pixel-table .col-name {
		width: 100%;
	}

	.pixel-table .col-shrink {
		width: 1%;
	}

	.pixel-table th,
	.pixel-table td {
		white-space: nowrap;
	}

	.pixel-table th {
		padding: 10px 14px;
	}

	.pixel-table td {
		padding: 12px 14px;
	}

	.slug-line code {
		font-size: 13.5px;
		font-weight: 600;
	}

	.ellipsis {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.metric-cell {
		font-weight: 600;
		font-size: 14px;
	}

	.badge.badge-idle {
		color: var(--muted);
	}

	.actions-cell {
		text-align: right;
	}

	.actions {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 6px;
	}

	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 30px;
		height: 30px;
		padding: 0;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--panel);
		color: var(--muted);
		cursor: pointer;
		transition:
			color 0.12s ease,
			border-color 0.12s ease,
			background 0.12s ease;
	}

	.icon-btn:hover {
		color: var(--text);
		border-color: var(--border-strong);
		background: var(--panel-2);
	}

	.icon-btn svg {
		width: 15px;
		height: 15px;
		fill: none;
		stroke: currentColor;
		stroke-width: 1.6;
		stroke-linecap: round;
		stroke-linejoin: round;
	}

	.readonly-value {
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 38px;
		padding: 8px 12px;
		border: 1px dashed var(--border);
		border-radius: 10px;
		background: var(--panel-2);
		color: var(--muted);
	}

	.readonly-value code {
		color: var(--text);
		font-size: 13px;
	}
</style>
