<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '#lib/api/client.js';
	import { session } from '#lib/stores/session.svelte.js';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		loading = true;

		try {
			await session.login(email, password);
			await goto('/app/');
		} catch (err) {
			error = err instanceof ApiError ? err.message : '登录失败，请稍后重试';
		} finally {
			loading = false;
		}
	}
</script>

<div class="auth-wrap">
	<form class="auth-card" onsubmit={submit}>
		<div class="brand" style="padding: 0">
			<span class="brand-mark">Y</span>
			<span>Yu 控制台</span>
		</div>

		{#if error}
			<div class="error">{error}</div>
		{/if}

		<label class="field">
			邮箱
			<input
				class="input"
				type="email"
				bind:value={email}
				required
				autocomplete="username"
				placeholder="you@example.com"
			/>
		</label>

		<label class="field">
			密码
			<input
				class="input"
				type="password"
				bind:value={password}
				required
				autocomplete="current-password"
				placeholder="至少 8 位"
			/>
		</label>

		<button class="btn btn-primary" type="submit" disabled={loading}>
			{loading ? '登录中…' : '登录'}
		</button>

		<p class="small muted">仅支持邀请制账号，请联系管理员获取邀请链接。</p>
	</form>
</div>
