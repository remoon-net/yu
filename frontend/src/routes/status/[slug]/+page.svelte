<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import type { PublicStatusResponse } from '#lib/api/types.js';

	const slug = $derived(page.params.slug ?? '');
	let data = $state<PublicStatusResponse | null>(null);
	let error = $state('');
	let needPassword = $state(false);
	let password = $state('');
	let unlocking = $state(false);

	$effect(() => {
		void load();
	});

	async function load() {
		error = '';
		try {
			data = await api.get<PublicStatusResponse>(`/api/public/status/${slug}`);
			needPassword = false;
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				needPassword = true;
				return;
			}
			error = err instanceof Error ? err.message : '加载失败';
		}
	}

	async function unlock(event: SubmitEvent) {
		event.preventDefault();
		unlocking = true;
		try {
			data = await api.get<PublicStatusResponse>(
				`/api/public/status/${slug}?password=${encodeURIComponent(password)}`
			);
			needPassword = false;
		} catch (err) {
			error = err instanceof ApiError ? err.message : '密码错误';
		} finally {
			unlocking = false;
		}
	}

	const statusText = $derived(
		data?.overall === 'down' ? '部分服务异常' : data?.overall === 'unknown' ? '等待数据' : '所有系统运行正常'
	);

	const groups = $derived.by(() => {
		const map = new Map<string, PublicStatusResponse['items']>();
		for (const item of data?.items ?? []) {
			const list = map.get(item.group_name) ?? [];
			list.push(item);
			map.set(item.group_name, list);
		}
		return [...map.entries()];
	});
</script>

<div class="auth-wrap" style="align-items: flex-start; padding-top: 48px">
	<div style="width: 100%; max-width: 760px" class="stack">
		{#if needPassword}
			<form class="auth-card" onsubmit={unlock}>
				<h1>需要访问密码</h1>
				{#if error}<div class="error">{error}</div>{/if}
				<label class="field">密码<input class="input" type="password" bind:value={password} required /></label>
				<button class="btn btn-primary" type="submit" disabled={unlocking}>
					{unlocking ? '验证中…' : '进入'}
				</button>
			</form>
		{:else if error}
			<div class="error">{error}</div>
		{:else if data}
			<div>
				<h1>{data.page.name}</h1>
				<p class="muted">{data.page.description}</p>
			</div>

			<div class="card" style="text-align: center">
				<div class="badge" class:ok={data.overall === 'operational'} style="font-size: 14px; padding: 8px 16px">
					{statusText}
				</div>
				<p class="small muted" style="margin-top: 10px">更新于 {new Date(data.updated_at).toLocaleString('zh-CN')}</p>
			</div>

			{#each groups as [groupName, items] (groupName)}
				<div class="card">
					<div class="card-head"><h2>{groupName}</h2></div>
					<table class="table">
						<thead><tr><th>服务</th><th>状态</th><th class="num">24h 可用率</th><th class="num">延迟</th></tr></thead>
						<tbody>
							{#each items as item (item.monitor_id)}
								<tr>
									<td>{item.name}</td>
									<td>
										<span class="badge" class:ok={item.status === 'up'}>
											{item.status === 'up' ? '正常' : item.status === 'down' ? '异常' : '待检测'}
										</span>
									</td>
									<td class="num">{item.uptime_24h.toFixed(2)}%</td>
									<td class="num">{item.latency_ms ?? '—'} ms</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/each}

			{#if data.incidents.length > 0}
				<div class="card">
					<div class="card-head"><h2>最近事件</h2></div>
					{#each data.incidents as incident (incident.id)}
						<div style="padding: 8px 0; border-bottom: 1px solid var(--border)">
							<div class="row">
								<strong>{incident.title}</strong>
								<span class="badge" class:ok={incident.resolved_at !== null}>{incident.resolved_at ? '已恢复' : '进行中'}</span>
							</div>
							<div class="small muted">{new Date(incident.started_at).toLocaleString('zh-CN')}</div>
						</div>
					{/each}
				</div>
			{/if}
		{:else}
			<p class="empty">加载中…</p>
		{/if}
	</div>
</div>
