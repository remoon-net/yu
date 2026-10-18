<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError, api } from '#lib/api/client.js';
	import type { Workspace } from '#lib/api/types.js';
	import { session } from '#lib/stores/session.svelte.js';

	let name = $state('');
	let slug = $state('');
	let error = $state('');
	let creating = $state(false);

	$effect(() => {
		if (session.loaded && session.workspaces.length > 0) {
			void goto(`/app/w/${session.workspaces[0].slug}/analytics/`, { replace: true });
		}
	});

	async function create(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		creating = true;

		try {
			const workspace = await api.post<Workspace>('/api/workspaces', {
				name,
				slug: slug.trim().toLowerCase()
			});
			await session.load(true);
			await goto(`/app/w/${workspace.slug}/analytics/`);
		} catch (err) {
			error = err instanceof ApiError ? err.message : '创建工作区失败';
		} finally {
			creating = false;
		}
	}
</script>

{#if session.loaded && session.workspaces.length === 0}
	<div class="auth-wrap">
		<form class="auth-card" onsubmit={create}>
			<h1>创建工作区</h1>
			<p class="small muted">工作区用于隔离站点、监控与成员。</p>

			{#if error}
				<div class="error">{error}</div>
			{/if}

			<label class="field">
				工作区名称
				<input class="input" bind:value={name} required maxlength={60} placeholder="例如：我的项目" />
			</label>

			<label class="field">
				工作区标识（可选）
				<input
					class="input"
					bind:value={slug}
					pattern={"[a-z0-9][a-z0-9\\-]{0,38}[a-z0-9]"}
					maxlength={40}
					placeholder="my-team"
				/>
			</label>

			<p class="small muted">路径会变成 /app/w/&lt;标识&gt;/；留空自动生成 ws-xxxxxx。</p>

			<button class="btn btn-primary" type="submit" disabled={creating}>
				{creating ? '创建中…' : '创建工作区'}
			</button>
		</form>
	</div>
{:else}
	<div class="boot">正在进入…</div>
{/if}
