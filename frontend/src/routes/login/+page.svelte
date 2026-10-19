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

<div class="auth">
	<section class="intro">
		<div class="grid-bg" aria-hidden="true"></div>
		<div class="intro-inner">
			<div class="logo">
				<span class="brand-mark">Y</span>
				<span>Yu</span>
			</div>
			<h1>轻量网站统计<br />与服务监控</h1>
			<p class="lead">
				PV / UV、来源与访客画像、实时访客、自定义事件，以及 HTTP / TCP / Ping / DNS 拨测与告警。
			</p>
			<ul class="features">
				<li>原始数据永久保留，无采样</li>
				<li>单二进制后端 + 纯静态前端</li>
				<li>公开看板与状态页</li>
				<li>短链与遥测像素</li>
			</ul>
		</div>
	</section>

	<section class="panel">
		<form class="form" onsubmit={submit}>
			<div class="form-head">
				<h2>登录控制台</h2>
				<p class="small muted">使用邀请制账号登录</p>
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

			<button class="btn btn-primary login-btn" type="submit" disabled={loading}>
				{loading ? '登录中…' : '登录'}
			</button>

			<p class="small muted footer-hint">仅支持邀请制账号，请联系管理员获取邀请链接。</p>
		</form>
	</section>
</div>

<style>
	.auth {
		min-height: 100vh;
		display: grid;
		grid-template-columns: minmax(0, 1.05fr) minmax(380px, 0.95fr);
	}

	.intro {
		position: relative;
		display: flex;
		align-items: center;
		overflow: hidden;
		padding: 64px clamp(28px, 7vw, 92px);
		background:
			radial-gradient(circle at 18% 12%, rgba(37, 99, 235, 0.14), transparent 46%),
			radial-gradient(circle at 85% 90%, rgba(99, 102, 241, 0.1), transparent 42%),
			var(--panel);
		border-right: 1px solid var(--border);
	}

	.grid-bg {
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(var(--border) 1px, transparent 1px),
			linear-gradient(90deg, var(--border) 1px, transparent 1px);
		background-size: 44px 44px;
		mask-image: radial-gradient(circle at 30% 35%, #000 10%, transparent 72%);
		opacity: 0.5;
	}

	.intro-inner {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 20px;
		max-width: 520px;
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 19px;
		font-weight: 650;
		letter-spacing: 0.02em;
	}

	.intro h1 {
		font-size: clamp(30px, 3.2vw, 42px);
		line-height: 1.2;
		letter-spacing: -0.03em;
	}

	.lead {
		max-width: 460px;
		font-size: 14px;
		line-height: 1.75;
		color: var(--muted);
	}

	.features {
		list-style: none;
		margin: 10px 0 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 10px 18px;
		font-size: 13px;
		color: var(--muted);
	}

	.features li {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.features li::before {
		content: '';
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--accent);
		opacity: 0.7;
		flex: none;
	}

	.panel {
		display: grid;
		place-items: center;
		padding: 32px;
	}

	.form {
		width: 100%;
		max-width: 360px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.form-head {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-bottom: 4px;
	}

	.form-head h2 {
		font-size: 21px;
		font-weight: 650;
		letter-spacing: -0.02em;
	}

	.login-btn {
		width: 100%;
		height: 38px;
		margin-top: 2px;
	}

	.footer-hint {
		margin-top: 2px;
	}

	@media (max-width: 900px) {
		.auth {
			grid-template-columns: minmax(0, 1fr);
			place-items: center;
			padding: 24px;
		}

		.intro {
			display: none;
		}

		.panel {
			padding: 0;
		}

		.form {
			max-width: 380px;
			padding: 28px;
			background: var(--panel);
			border: 1px solid var(--border);
			border-radius: 16px;
			box-shadow: var(--shadow-pop);
		}
	}
</style>
