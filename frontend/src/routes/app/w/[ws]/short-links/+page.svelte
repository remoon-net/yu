<script lang="ts">
	import { page } from '$app/state';
	import type { EChartsCoreOption } from 'echarts/core';
	import QRCode from 'qrcode';
	import { ApiError, api } from '#lib/api/client.js';
	import { shortLinkApi } from '#lib/api/monitors.js';
	import type { ShortLink, ShortLinkStats, ShortLinkStatsResponse, Site } from '#lib/api/types.js';
	import BarList from '#lib/components/BarList.svelte';
	import Chart from '#lib/components/Chart.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';
	import { formatChartTime, formatDateTime, formatNumber } from '#lib/utils/format.js';

	const slug = $derived(page.params.ws ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	const detailRanges = [
		{ value: '7d', label: '近 7 天' },
		{ value: '30d', label: '近 30 天' },
		{ value: '90d', label: '近 90 天' }
	];

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

	function origin() {
		return window.location.origin;
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
		const ok = await confirmDialog({
			title: '删除短链',
			message: `确定删除短链「${link.slug}」？删除后该链接立即失效。`,
			confirmText: '删除'
		});
		if (!ok) return;

		try {
			await shortLinkApi.remove(workspace.id, siteId, link.id);
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

	const detailOption = $derived<EChartsCoreOption>({
		grid: { left: 44, right: 16, top: 34, bottom: 28 },
		tooltip: { trigger: 'axis' },
		legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 11 } },
		xAxis: {
			type: 'category',
			boundaryGap: false,
			data: (detail?.stats.series ?? []).map((point) =>
				formatChartTime(point.bucket, timezone, detail?.range.interval ?? 'day')
			),
			axisLine: { lineStyle: { opacity: 0.25 } },
			axisLabel: { fontSize: 11, hideOverlap: true }
		},
		yAxis: {
			type: 'value',
			splitLine: { lineStyle: { opacity: 0.12 } },
			axisLabel: { fontSize: 11 }
		},
		series: [
			{
				name: '点击',
				type: 'line',
				smooth: true,
				showSymbol: false,
				areaStyle: { opacity: 0.16 },
				data: (detail?.stats.series ?? []).map((point) => point.pv)
			},
			{
				name: '访客',
				type: 'line',
				smooth: true,
				showSymbol: false,
				data: (detail?.stats.series ?? []).map((point) => point.uv)
			}
		]
	});
</script>

<div class="page-head">
	<div>
		<h1>短链</h1>
		<p class="small muted">创建可追踪的短链接，点击会计入站点统计。</p>
	</div>
	<div class="row">
		{#if sites.length > 1}
			<select class="select" bind:value={siteId} style="min-width: 160px">
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
	<div class="card" style="overflow-x: auto">
		<table class="table">
			<thead>
				<tr>
					<th>短链</th>
					<th>目标地址</th>
					<th>目标域名</th>
					<th class="num">点击</th>
					<th class="num">访客</th>
					<th>状态</th>
					<th>最近点击</th>
					<th class="num"></th>
				</tr>
			</thead>
			<tbody>
				{#each links as link (link.id)}
					{@const item = statOf(link)}
					<tr>
						<td>
							<div><code>/s/{link.slug}</code></div>
							{#if link.title}<div class="small muted">{link.title}</div>{/if}
						</td>
						<td class="small muted" style="max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap" title={link.target_url}>
							{link.target_url}
						</td>
						<td class="small">
							<span class="host-name">{targetHost(link.target_url)}</span>
							<span class="badge" class:ok={isInternalTarget(link)}>{isInternalTarget(link) ? '站内' : '站外'}</span>
						</td>
						<td class="num">{item?.clicks ?? 0}</td>
						<td class="num">{item?.visitors ?? 0}</td>
						<td>
							<span class="badge" class:ok={statusOf(link) === '启用中'}>{statusOf(link)}</span>
						</td>
						<td class="small muted">
							{item?.last_click ? new Date(item.last_click).toLocaleString('zh-CN') : '—'}
						</td>
						<td class="num">
							<button class="btn btn-sm" type="button" onclick={() => openDetail(link)}>详情</button>
							<button class="btn btn-sm" type="button" onclick={() => copy(link)}>复制</button>
							<button class="btn btn-sm" type="button" onclick={() => openQr(link)}>二维码</button>
							{#if manageable}
								<button class="btn btn-sm" type="button" onclick={() => openEdit(link)}>编辑</button>
								<button class="btn btn-sm btn-danger" type="button" onclick={() => remove(link)}>删除</button>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<p class="small muted">短链地址基于当前域名：{origin()}/s/&lt;标识&gt;</p>
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
		<div class="detail-top">
			<div class="detail-metrics">
				<div class="metric">
					<span class="small muted">点击</span>
					<strong>{formatNumber(detail?.stats.clicks ?? 0)}</strong>
				</div>
				<div class="metric">
					<span class="small muted">独立访客</span>
					<strong>{formatNumber(detail?.stats.visitors ?? 0)}</strong>
				</div>
				<div class="metric">
					<span class="small muted">最近点击</span>
					<strong class="small">
						{detail?.stats.last_click ? formatDateTime(detail.stats.last_click, timezone) : '—'}
					</strong>
				</div>
			</div>
			<div class="row" style="gap: 6px">
				{#each detailRanges as item (item.value)}
					<button
						class="btn btn-sm"
						class:btn-primary={detailRange === item.value}
						type="button"
						onclick={() => setDetailRange(item.value)}
					>
						{item.label}
					</button>
				{/each}
			</div>
		</div>

		<div class="detail-target small muted">
			<span>目标：</span>
			<code title={detailTarget.target_url}>{detailTarget.target_url}</code>
			<span class="badge" class:ok={isInternalTarget(detailTarget)}>
				{isInternalTarget(detailTarget) ? '站内' : '站外'}
			</span>
		</div>

		{#if detailError}
			<div class="error" style="margin: 12px 0">{detailError}</div>
		{:else if detailLoading && !detail}
			<p class="empty">加载中…</p>
		{:else if detail}
			<div class="detail-chart">
				<Chart option={detailOption} height="240px" />
			</div>
			{#if detail.stats.series.length === 0}
				<p class="small muted">该区间内还没有点击记录。</p>
			{/if}

			<div class="detail-grid">
				<section class="detail-tile">
					<h3>来源</h3>
					<BarList items={detail.stats.referrers} />
				</section>
				<section class="detail-tile">
					<h3>渠道</h3>
					<BarList items={detail.stats.channels} />
				</section>
				<section class="detail-tile">
					<h3>UTM 活动</h3>
					<BarList items={detail.stats.utm_campaign} emptyText="未带活动参数" />
				</section>
				<section class="detail-tile">
					<h3>设备</h3>
					<BarList items={detail.stats.device} />
				</section>
				<section class="detail-tile">
					<h3>UTM 来源</h3>
					<BarList items={detail.stats.utm_source} emptyText="未带来源参数" />
				</section>
				<section class="detail-tile">
					<h3>浏览器</h3>
					<BarList items={detail.stats.browser} />
				</section>
			</div>

			<h3 style="margin: 20px 0 8px; font-size: 14px">最近点击</h3>
			{#if detail.stats.recent.length === 0}
				<p class="empty">暂无点击明细。</p>
			{:else}
				<div style="overflow-x: auto">
					<table class="table">
						<thead>
							<tr>
								<th>时间</th>
								<th>来源</th>
								<th>渠道</th>
								<th>UTM</th>
								<th>设备</th>
							</tr>
						</thead>
						<tbody>
							{#each detail.stats.recent as click, index (index)}
								<tr>
									<td class="small muted">{formatDateTime(click.created_at, timezone)}</td>
									<td class="small">{click.referrer_host || '(direct)'}</td>
									<td class="small">{click.channel || '—'}</td>
									<td class="small">{click.utm_campaign || click.utm_source || '—'}</td>
									<td class="small">
										{[click.browser, click.os, click.device].filter(Boolean).join(' · ') || '—'}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}
	{/if}
</Modal>

<style>
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

	.detail-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}

	.detail-metrics {
		display: flex;
		gap: 26px;
		flex-wrap: wrap;
	}

	.metric {
		display: grid;
		gap: 2px;
	}

	.metric strong {
		font-size: 20px;
		line-height: 1.2;
	}

	.detail-target {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 12px;
		min-width: 0;
	}

	.detail-target code {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.detail-chart {
		margin-top: 10px;
	}

	.detail-grid {
		display: grid;
		gap: 12px;
		grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
		margin-top: 6px;
	}

	.detail-tile {
		padding: 12px 14px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--panel-2);
	}

	.detail-tile h3 {
		margin: 0 0 8px;
		font-size: 13px;
		font-weight: 600;
		color: var(--muted);
	}

	.host-name {
		margin-right: 6px;
	}
</style>
