<script lang="ts">
	import { HTTP_METHODS, type HttpFormState } from '#lib/utils/httpConfig.js';

	let { form = $bindable() }: { form: HttpFormState } = $props();

	const allowsBody = $derived(!['GET', 'HEAD'].includes(form.method));
</script>

<div class="http-options">
	<div class="grid-2">
		<label class="field">请求方法
			<select class="select" bind:value={form.method}>
				{#each HTTP_METHODS as method (method)}
					<option value={method}>{method}</option>
				{/each}
			</select>
		</label>
		<label class="field">期望状态码
			<input
				class="input"
				bind:value={form.expectedStatus}
				placeholder="200 或 200-299（留空默认 200-299）"
			/>
		</label>
	</div>

	<label class="field">自定义 Header
		<textarea
			class="input"
			bind:value={form.headers}
			rows="3"
			spellcheck="false"
			placeholder={'Authorization: Bearer xxx\nAccept: application/json'}
		></textarea>
		<span class="hint">每行一个，格式为「名称: 值」，空行与 # 开头的行会被忽略。</span>
	</label>

	<details class="advanced">
		<summary>高级选项（请求体、断言、重定向）</summary>

		<div class="grid-2 advanced-grid">
			{#if allowsBody}
				<label class="field wide">请求体
					<textarea
						class="input"
						bind:value={form.body}
						rows="3"
						spellcheck="false"
						placeholder={'{"ping":true}'}
					></textarea>
					<span class="hint">JSON 请求体未指定 Content-Type 时会自动补 application/json。</span>
				</label>
			{/if}

			<label class="field">响应体包含
				<input class="input" bind:value={form.bodyContains} placeholder="例如：ok" />
			</label>
			<label class="field">响应体不包含
				<input class="input" bind:value={form.bodyNotContains} placeholder="例如：error" />
			</label>
			<label class="field">JSONPath
				<input class="input" bind:value={form.jsonPath} placeholder="例如：$.status" />
			</label>
			<label class="field">JSONPath 期望值
				<input class="input" bind:value={form.jsonEquals} placeholder="例如：ok、42、true" />
			</label>
			<label class="field">响应耗时阈值（毫秒）
				<input class="input" type="number" min={1} bind:value={form.maxLatencyMS} placeholder="例如：2000" />
			</label>
			<label class="field">最大重定向次数
				<input class="input" type="number" min={1} max={20} bind:value={form.maxRedirects} />
			</label>
		</div>

		<label class="row follow">
			<input type="checkbox" bind:checked={form.followRedirects} />
			跟随重定向
		</label>
	</details>
</div>

<style>
	.http-options {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	textarea.input {
		height: auto;
		min-height: 76px;
		padding: 8px 11px;
		resize: vertical;
		font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
		font-size: 12.5px;
		line-height: 1.55;
	}

	.hint {
		font-size: 12px;
		color: var(--muted);
	}

	.advanced {
		border: 1px solid var(--border);
		border-radius: 9px;
		padding: 10px 12px;
	}

	.advanced summary {
		cursor: pointer;
		font-size: 13px;
		color: var(--muted);
		user-select: none;
	}

	.advanced-grid {
		margin-top: 14px;
	}

	.field.wide {
		grid-column: 1 / -1;
	}

	.follow {
		margin-top: 12px;
		gap: 8px;
		font-size: 13px;
	}
</style>
