<script lang="ts">
	import { page } from '$app/state';
	import QRCode from 'qrcode';
	import { ApiError, api } from '#lib/api/client.js';
	import { shortLinkApi } from '#lib/api/monitors.js';
	import type { ShortLink, ShortLinkStats, ShortLinkStatsResponse, Site } from '#lib/api/types.js';
	import HitDetail from '#lib/components/HitDetail.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';
	import { formatDateTime, formatNumber } from '#lib/utils/format.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let sites = $state<Site[]>([]);
	let siteId = $state('');
	let links = $state<ShortLink[]>([]);
	let stats = $state<Record<string, ShortLinkStats>>({});
	let error = $state('');
	let notice = $state('');

	let showForm = $state(false);
	let editing = $state<ShortLink | null>(null);
	let formTarget = $state('');
	let formSlug = $state('');
	let formTitle = $state('');
	let formEnabled = $state(true);
	let formExpires = $state('');
	let formError = $state('');
	let saving = $state(false);

	let qrTarget = $state<ShortLink | null>(null);
	let qrSvg = $state('');

	let detailTarget = $state<ShortLink | null>(null);
	let detail = $state<ShortLinkStatsResponse | null>(null);
	let detailRange = $state('30d');
	let detailLoading = $state(false);
	let detailError = $state('');

	const currentSite = $derived(sites.find((site) => site.id === siteId));
	const timezone = $derived(currentSite?.timezone || 'UTC');

	$effect(() => {
		if (!workspace) return;
		void loadSites(workspace.id);
	});

	$effect(() => {
		if (!workspace || !siteId) return;
		void loadLinks(workspace.id, siteId);
	});

	async function loadSites(workspaceId: string) {
		try {
			sites = await api.get<Site[]>(`/api/workspaces/${workspaceId}/sites`);
			if (!siteId && sites.length > 0) siteId = sites[0].id;
		} catch (err) {
			error = err instanceof Error ? err.message : '加载站点失败';
		}
	}

	async function loadLinks(workspaceId: string, currentSiteId: string) {
		try {
			const data = await shortLinkApi.list(workspaceId, currentSiteId);
			links = data.links ?? [];
			stats = Object.fromEntries((data.stats ?? []).map((item) => [item.slug, item]));
		} catch (err) {
			error = err instanceof Error ? err.message : '加载短链失败';
		}
	}

	function shortURL(linkSlug: string) {
		return `${window.location.origin}/s/${linkSlug}`;
	}

	function targetPath(raw: string) {
		try {
			const url = new URL(raw);
			return `${url.pathname}${url.search}` || '/';
		} catch {
			return raw;
		}
	}

	function statOf(link: ShortLink): ShortLinkStats | undefined {
		return stats[link.slug];
	}

	function statusOf(link: ShortLink) {
		if (!link.enabled) return '已停用';
		if (link.expires_at && new Date(link.expires_at) < new Date()) return '已过期';
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
		formTarget = '';
		formSlug = '';
		formTitle = '';
		formEnabled = true;
		formExpires = '';
		formError = '';
		showForm = true;
	}

	function openEdit(link: ShortLink) {
		editing = link;
		formTarget = link.target_url;
		formSlug = link.slug;
		formTitle = link.title;
		formEnabled = link.enabled;
		formExpires = toLocalInput(link.expires_at);
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
			target_url: formTarget.trim(),
			title: formTitle.trim(),
			enabled: formEnabled,
			expires_at: formExpires ? new Date(formExpires).toISOString() : null
		};

		try {
			if (editing) {
				await shortLinkApi.update(workspace.id, siteId, editing.id, body);
				notice = '短链已更新';
			} else {
				await shortLinkApi.create(workspace.id, siteId, body);
				notice = '短链已创建';
			}
			showForm = false;
			editing = null;
			await loadLinks(workspace.id, siteId);
		} catch (err) {
			formError = err instanceof ApiError ? err.message : '保存短链失败';
		} finally {
			saving = false;
		}
	}

	async function remove(link: ShortLink) {
		if (!workspace || !siteId) return;

		// 从编辑弹窗触发时，先关闭编辑弹窗再弹确认，取消可返回。
		const fromForm = editing?.id === link.id;
		if (fromForm) showForm = false;

		const ok = await confirmDialog({
			title: '删除短链',
			message: `确定删除短链「${link.slug}」？删除后该链接立即失效。`,
			confirmText: '删除'
		});

		if (!ok) {
			if (fromForm) showForm = true;
			return;
		}

		try {
			await shortLinkApi.remove(workspace.id, siteId, link.id);
			editing = null;
			notice = '短链已删除';
			await loadLinks(workspace.id, siteId);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除短链失败';
		}
	}

	async function copy(link: ShortLink) {
		try {
			await navigator.clipboard.writeText(shortURL(link.slug));
			notice = '短链地址已复制';
		} catch {
			error = '复制失败，请手动选择链接';
		}
	}

	async function openQr(link: ShortLink) {
		qrTarget = link;
		qrSvg = await QRCode.toString(shortURL(link.slug), { type: 'svg', margin: 1, width: 240 });
	}

	async function downloadQr() {
		if (!qrTarget) return;
		const dataUrl = await QRCode.toDataURL(shortURL(qrTarget.slug), { margin: 1, width: 640 });
		const a = document.createElement('a');
		a.href = dataUrl;
		a.download = `short-link-${qrTarget.slug}.png`;
		a.click();
	}

	function normalizeDomain(raw: string) {
		const value = raw.trim().toLowerCase().replace(/^https?:\/\//, '');
		return value.split('/')[0];
	}

	function targetHost(raw: string) {
		try {
			return new URL(raw).host.toLowerCase();
		} catch {
			return raw;
		}
	}

	function isInternalTarget(link: ShortLink) {
		const domain = normalizeDomain(currentSite?.domain ?? '');
		if (!domain) return false;
		const host = targetHost(link.target_url);
		return host === domain || host.endsWith(`.${domain}`);
	}

	async function openDetail(link: ShortLink) {
		detailTarget = link;
		detail = null;
		detailError = '';
		await loadDetail();
	}

	async function loadDetail() {
		if (!workspace || !siteId || !detailTarget) return;
		detailLoading = true;
		detailError = '';
		try {
			detail = await shortLinkApi.stats(workspace.id, siteId, detailTarget.id, detailRange);
		} catch (err) {
			detailError = err instanceof Error ? err.message : '加载短链统计失败';
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
		<h1>短链</h1>
		<p class="small muted">创建可追踪的短链接，点击会计入所选站点的统计。</p>
	</div>
	<div class="row">
		{#if sites.length > 1}
			<select class="select site-select" bind:value={siteId}>
				{#each sites as site (site.id)}<option value={site.id}>{site.name}</option>{/each}
			</select>
		{/if}
		{#if manageable}
			<button class="btn btn-primary" type="button" onclick={openCreate} disabled={sites.length === 0}>新建短链</button>
		{/if}
	</div>
</div>

{#if error}<div class="error" style="margin-bottom: 16px">{error}</div>{/if}
{#if notice}<div class="badge ok" style="margin-bottom: 16px">{notice}</div>{/if}

{#if sites.length === 0}
	<div class="card"><p class="empty">还没有站点，请先在「站点设置」中添加站点。</p></div>
{:else if links.length === 0}
	<div class="card"><p class="empty">还没有短链。</p></div>
{:else}
	<div class="card table-card">
		<table class="table link-table">
			<thead>
				<tr>
					<th class="col-link">短链</th>
					<th class="col-target">目标</th>
					<th class="num col-shrink">点击</th>
					<th class="num col-shrink">访客</th>
					<th class="col-shrink">状态</th>
					<th class="col-shrink">最近点击</th>
					<th class="col-shrink"></th>
				</tr>
			</thead>
			<tbody>
				{#each links as link (link.id)}
					{@const item = statOf(link)}
					<tr>
						<td>
							<div class="slug-line"><code>/s/{link.slug}</code></div>
							{#if link.title}<div class="small muted ellipsis" title={link.title}>{link.title}</div>{/if}
						</td>
						<td class="target-cell">
							<div class="target-line">
								<span class="host">{targetHost(link.target_url)}</span>
								<span class="badge" class:ok={isInternalTarget(link)}>
									{isInternalTarget(link) ? '站内' : '站外'}
								</span>
							</div>
							<div class="small muted ellipsis" title={link.target_url}>{targetPath(link.target_url)}</div>
						</td>
						<td class="num metric-cell">{formatNumber(item?.clicks ?? 0)}</td>
						<td class="num metric-cell">{formatNumber(item?.visitors ?? 0)}</td>
						<td>
							<span
								class="badge"
								class:ok={statusOf(link) === '启用中'}
								class:badge-idle={statusOf(link) !== '启用中'}
							>
								{statusOf(link)}
							</span>
						</td>
						<td class="small muted">
							{item?.last_click ? formatDateTime(item.last_click, timezone) : '—'}
						</td>
						<td class="actions-cell">
							<div class="actions">
							<button class="btn btn-sm" type="button" onclick={() => openDetail(link)}>详情</button>
							<button class="icon-btn" type="button" title="复制短链" aria-label="复制短链" onclick={() => copy(link)}>
								<svg viewBox="0 0 24 24" aria-hidden="true">
									<rect x="9" y="9" width="11" height="11" rx="2" />
									<path d="M5 15V5a2 2 0 0 1 2-2h10" />
								</svg>
							</button>
							<button class="icon-btn" type="button" title="二维码" aria-label="二维码" onclick={() => openQr(link)}>
								<svg viewBox="0 0 24 24" aria-hidden="true">
									<rect x="3" y="3" width="7" height="7" rx="1" />
									<rect x="14" y="3" width="7" height="7" rx="1" />
									<rect x="3" y="14" width="7" height="7" rx="1" />
									<path d="M14 14h3v3h-3zM20 14v3M14 20h3M20 20h1" />
								</svg>
							</button>
							{#if manageable}
								<button class="icon-btn" type="button" title="编辑" aria-label="编辑" onclick={() => openEdit(link)}>
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

<Modal open={showForm} title={editing ? '编辑短链' : '新建短链'} onclose={() => (showForm = false)}>
	<form onsubmit={submit}>
		<div class="grid-2">
			<label class="field">
				目标地址
				<input class="input" bind:value={formTarget} required placeholder="https://example.com/promo" />
			</label>
			<label class="field">
				短链标识{editing ? '' : '（可选）'}
				{#if editing}
					<div class="readonly-value"><code>/s/{formSlug}</code><span class="small muted">创建后不可修改</span></div>
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
			<label class="field">备注<input class="input" bind:value={formTitle} maxlength={80} placeholder="例如：双十一活动" /></label>
			<label class="field">过期时间（可选）<input class="input" type="datetime-local" bind:value={formExpires} /></label>
		</div>
		<label class="row" style="margin-top: 14px; gap: 8px; font-size: 13px">
			<input type="checkbox" bind:checked={formEnabled} /> 启用该短链
		</label>
		{#if formError}<div class="error" style="margin-top: 12px">{formError}</div>{/if}
		<div class="modal-actions">
			{#if editing}
				<button
					class="btn btn-danger"
					type="button"
					style="margin-right: auto"
					onclick={() => remove(editing!)}
				>
					删除短链
				</button>
			{/if}
			<button class="btn" type="button" onclick={() => (showForm = false)}>取消</button>
			<button class="btn btn-primary" type="submit" disabled={saving}>{saving ? '保存中…' : '保存'}</button>
		</div>
	</form>
</Modal>

<Modal open={!!qrTarget} title="短链二维码" size="sm" onclose={() => (qrTarget = null)}>
	{#if qrTarget}
		<div class="qr-wrap">
			<div class="qr-image">{@html qrSvg}</div>
			<code>{shortURL(qrTarget.slug)}</code>
		</div>
		<div class="modal-actions">
			<button class="btn" type="button" onclick={() => (qrTarget = null)}>关闭</button>
			<button class="btn btn-primary" type="button" onclick={downloadQr}>下载 PNG</button>
		</div>
	{/if}
</Modal>

<Modal
	open={!!detailTarget}
	title={`短链详情 · /s/${detailTarget?.slug ?? ''}`}
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
			target={{ text: detailTarget.target_url, internal: isInternalTarget(detailTarget) }}
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

	.link-table {
		width: 100%;
	}

	.link-table .col-link {
		width: 200px;
	}

	.link-table .col-target {
		width: 100%;
	}

	.link-table .col-shrink {
		width: 1%;
	}

	.link-table th,
	.link-table td {
		white-space: nowrap;
	}

	.link-table th {
		padding: 10px 14px;
	}

	.link-table td {
		padding: 12px 14px;
	}

	.slug-line code {
		font-size: 13.5px;
		font-weight: 600;
	}

	.target-cell {
		max-width: 320px;
	}

	.target-line {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.host {
		font-size: 13px;
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

	.qr-wrap {
		display: grid;
		justify-items: center;
		gap: 10px;
		padding: 8px 0;
	}

	.qr-image {
		padding: 10px;
		border-radius: 12px;
		background: #fff;
		box-shadow: 0 10px 30px -18px rgba(15, 23, 42, 0.45);
	}

	.qr-image :global(svg) {
		display: block;
		width: 220px;
		height: 220px;
	}
</style>
