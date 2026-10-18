<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError, api } from '#lib/api/client.js';
	import type { SessionPayload } from '#lib/api/types.js';
	import { session } from '#lib/stores/session.svelte.js';

	const token = $derived(page.params.token ?? '');

	let password = $state('');
	let name = $state('');
	let error = $state('');
	let loading = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		loading = true;

		try {
			const data = await api.post<SessionPayload>('/api/auth/invite/accept', {
				token,
				password,
				name
			});
			session.apply(data);
			await goto('/app');
		} catch (err) {
			error = err instanceof ApiError ? err.message : '加入失败，请稍后重试';
		} finally {
			loading = false;
		}
	}
</script>

<div class="auth-wrap">
	<form class="auth-card" onsubmit={submit}>
		<div class="brand" style="padding: 0">
			<span class="brand-mark">Y</span>
			<span>加入工作区</span>
		</div>

		{#if error}
			<div class="error">{error}</div>
		{/if}

		<label class="field">
			你的名字
			<input class="input" bind:value={name} placeholder="留空则使用邮箱前缀" />
		</label>

		<label class="field">
			设置密码
			<input
				class="input"
				type="password"
				bind:value={password}
				required
				minlength={8}
				autocomplete="new-password"
				placeholder="至少 8 位"
			/>
		</label>

		<button class="btn btn-primary" type="submit" disabled={loading || !token}>
			{loading ? '提交中…' : '接受邀请'}
		</button>

		{#if !token}
			<p class="error">邀请链接缺少 token。</p>
		{/if}
	</form>
</div>
