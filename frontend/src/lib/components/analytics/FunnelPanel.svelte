<script lang="ts">
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { FunnelResponse, FunnelStepInput, StatsFilter } from '#lib/api/types.js';
	import { formatNumber } from '#lib/utils/format.js';

	let {
		workspaceId,
		siteId,
		range,
		filter
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
		filter?: StatsFilter;
	} = $props();

	let steps = $state<FunnelStepInput[]>([
		{ type: 'page', value: '/' },
		{ type: 'event', value: '' }
	]);
	let result = $state<FunnelResponse | null>(null);
	let loading = $state(false);
	let error = $state('');

	const filterKey = $derived(JSON.stringify(filter ?? {}));

	$effect(() => {
		// 时间范围或筛选条件变化后旧结果不再可信。
		const span = range;
		const key = filterKey;
		void span;
		void key;
		result = null;
	});

	function addStep() {
		if (steps.length >= 5) return;
		steps = [...steps, { type: 'event', value: '' }];
	}

	function removeStep(index: number) {
		if (steps.length <= 2) return;
		steps = steps.filter((_, i) => i !== index);
	}

	async function analyze() {
		error = '';

		for (const step of steps) {
			if (!step.value.trim()) {
				error = '请填写每个步骤的页面路径或事件名';
				return;
			}
		}

		loading = true;
		try {
			result = await analyticsApi.funnel(workspaceId, siteId, range, steps, filter);
		} catch (err) {
			error = err instanceof Error ? err.message : '分析漏斗失败';
		} finally {
			loading = false;
		}
	}

	function stepLabel(step: FunnelStepInput, index: number): string {
		const kind = step.type === 'event' ? '事件' : '页面';
		return `${index + 1}. ${kind} · ${step.value}`;
	}

	const baseVisitors = $derived(result?.steps[0]?.visitors ?? 0);
</script>

<div class="card">
	<div class="card-head">
		<h2>漏斗分析</h2>
		<span class="small muted">按顺序计算完成每一步的访客</span>
	</div>

	<div class="stack" style="gap: 10px">
		{#each steps as step, index (index)}
			<div class="step-row">
				<span class="step-index">{index + 1}</span>
				<select class="select step-type" bind:value={step.type}>
					<option value="page">页面访问</option>
					<option value="event">事件</option>
				</select>
				<input
					class="input"
					bind:value={step.value}
					placeholder={step.type === 'event' ? '例如：signup' : '例如：/pricing'}
				/>
				<button
					class="btn btn-sm"
					type="button"
					disabled={steps.length <= 2}
					onclick={() => removeStep(index)}
				>
					移除
				</button>
			</div>
		{/each}
	</div>

	{#if error}
		<div class="error" style="margin-top: 12px">{error}</div>
	{/if}

	<div class="row" style="margin-top: 14px">
		<button class="btn btn-sm" type="button" disabled={steps.length >= 5} onclick={addStep}>
			添加步骤
		</button>
		<button class="btn btn-primary" type="button" disabled={loading} onclick={analyze}>
			{loading ? '分析中…' : '分析'}
		</button>
	</div>
</div>

{#if result}
	<div class="card" style="margin-top: 16px">
		<div class="card-head">
			<h2>漏斗结果</h2>
			<span class="small muted">完成第一步的访客为 100%</span>
		</div>

		<div class="funnel-list">
			{#each result.steps as step, index (index)}
				{@const previous = index > 0 ? result.steps[index - 1].visitors : step.visitors}
				{@const width = baseVisitors > 0 ? (step.visitors / baseVisitors) * 100 : 0}
				{@const drop = previous > 0 ? ((previous - step.visitors) / previous) * 100 : 0}
				<div class="funnel-step">
					<div class="funnel-row">
						<span class="funnel-label">{stepLabel(step, index)}</span>
						<span class="funnel-count">{formatNumber(step.visitors)} 访客</span>
					</div>
					<div class="funnel-track">
						<div class="funnel-fill" style:width={`${Math.max(width, 1)}%`}></div>
					</div>
					<div class="funnel-meta small muted">
						<span>{step.conversion_rate.toFixed(2)}% 转化</span>
						{#if index > 0}
							<span>较上一步流失 {drop.toFixed(1)}%</span>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</div>
{/if}

<style>
	.step-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.step-index {
		width: 22px;
		text-align: center;
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}

	.step-type {
		width: 110px;
		flex: none;
	}

	.funnel-list {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.funnel-row {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 6px;
	}

	.funnel-label {
		font-size: 13px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.funnel-count {
		font-size: 13px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}

	.funnel-track {
		height: 10px;
		border-radius: 999px;
		background: var(--panel-2);
		overflow: hidden;
	}

	.funnel-fill {
		height: 100%;
		border-radius: 999px;
		background: linear-gradient(90deg, #3b82f6, #2563eb);
	}

	.funnel-meta {
		display: flex;
		justify-content: space-between;
		margin-top: 4px;
	}

	@media (max-width: 640px) {
		.step-row {
			flex-wrap: wrap;
		}

		.step-row .input {
			flex: 1 1 100%;
		}
	}
</style>
