<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import type { Site } from '#lib/api/types.js';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { canManage, session } from '#lib/stores/session.svelte.js';

	const slug = $derived(page.params.ws ?? '');
	const siteId = $derived(page.params.siteId ?? '');
	const workspace = $derived(session.findWorkspace(slug));
	const manageable = $derived(canManage(workspace?.role));

	let site = $state<Site | null>(null);
	let error = $state('');
	let notice = $state('');
	let saving = $state(false);
	let deleting = $state(false);
	let copied = $state('');

	let name = $state('');
	let domain = $state('');
	let timezone = $state('UTC');

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
		if (!workspace || !siteId) return;

		const workspaceId = workspace.id;
		let cancelled = false;

		error = '';
		site = null;

		api
			.get<Site>(`/api/workspaces/${workspaceId}/sites/${siteId}`)
			.then((result) => {
				if (cancelled) return;
				site = result;
				name = result.name;
				domain = result.domain;
				timezone = result.timezone;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载站点失败';
			});

		return () => {
			cancelled = true;
		};
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!site || !workspace) return;

		saving = true;
		error = '';
		notice = '';

		try {
			site = await api.patch<Site>(`/api/workspaces/${workspace.id}/sites/${site.id}`, {
				name,
				domain,
				timezone
			});
			notice = '已保存';
		} catch (err) {
			error = err instanceof ApiError ? err.message : '保存失败';
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!site || !workspace) return;
		const ok = await confirmDialog({
			title: '删除站点',
			message: `确定删除站点「${site.name}」？该站点的统计数据会被一并删除。`,
			confirmText: '删除'
		});
		if (!ok) return;

		deleting = true;
		try {
			await api.delete(`/api/workspaces/${workspace.id}/sites/${site.id}`);
			await goto(`/app/w/${slug}/analytics/`);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除失败';
		} finally {
			deleting = false;
		}
	}

	async function copy(text: string) {
		try {
			await navigator.clipboard.writeText(text);
			copied = text;
			setTimeout(() => (copied = ''), 1500);
		} catch {
			error = '复制失败，请手动选择文本';
		}
	}
</script>

<div class="page-head">
	<div>
		<h1>站点设置</h1>
		<p class="small muted">
			<a href={`/app/w/${slug}/analytics/`}>返回站点列表</a>
		</p>
	</div>

	{#if site}
		<a class="btn btn-sm" href={`/app/w/${slug}/analytics/${site.id}/`}>查看看板</a>
	{/if}
</div>

{#if error}
	<div class="error" style="margin-bottom: 16px">{error}</div>
{/if}
{#if notice}
	<div class="badge ok" style="margin-bottom: 16px">{notice}</div>
{/if}

{#if site && workspace}
	{@const snippet = site.install_snippet}
	<div class="stack">
		<form class="card" onsubmit={save}>
			<div class="card-head"><h2>基础信息</h2></div>

			<div class="grid-2">
				<label class="field">
					站点名称
					<input class="input" bind:value={name} required maxlength={60} disabled={!manageable} />
				</label>

				<label class="field">
					域名
					<input class="input" bind:value={domain} placeholder="example.com" disabled={!manageable} />
				</label>

				<label class="field">
					时区
					<select class="select timezone-select" bind:value={timezone} disabled={!manageable}>
						{#each timezones as zone (zone)}
							<option value={zone}>{zone}</option>
						{/each}
					</select>
				</label>
			</div>

			{#if manageable}
				<div class="row" style="margin-top: 14px">
					<button class="btn btn-primary" type="submit" disabled={saving}>
						{saving ? '保存中…' : '保存'}
					</button>
				</div>
			{/if}
		</form>

		<div class="card">
			<div class="card-head">
				<h2>安装采集脚本</h2>
				<span class="small muted">放在 &lt;head&gt; 或 &lt;body&gt; 末尾</span>
			</div>

			<div class="copy-box">
				{snippet}
				<button class="btn btn-sm" type="button" onclick={() => copy(snippet)}>
					{copied === snippet ? '已复制' : '复制'}
				</button>
			</div>

			<div class="small muted" style="margin-top: 12px">
				脚本地址：<code>{site.script_url}</code><br />
				上报地址：<code>{site.endpoint}</code><br />
				分享链接：<code>{site.share_url}</code>
			</div>

			<div class="small muted" style="margin-top: 12px">
				自定义事件：<code>window.yu.event('name', &#123; level: 'pro' &#125;)</code>
			</div>
		</div>

		{#if manageable}
			<div class="card">
				<div class="card-head"><h2>危险操作</h2></div>
				<p class="small muted" style="margin-bottom: 12px">
					删除站点会同时删除其全部事件与会话数据，且无法恢复。
				</p>
				<button class="btn btn-danger" type="button" disabled={deleting} onclick={remove}>
					{deleting ? '删除中…' : '删除站点'}
				</button>
			</div>
		{/if}
	</div>
{:else if !error}
	<p class="empty">加载中…</p>
{/if}
