<script lang="ts">
	import type { CountStat } from '#lib/api/types.js';
	import { formatNumber } from '#lib/utils/format.js';

	let {
		items,
		emptyText = '暂无数据'
	}: {
		items: CountStat[];
		emptyText?: string;
	} = $props();

	const max = $derived(Math.max(1, ...items.map((item) => item.count)));
</script>

{#if items.length === 0}
	<p class="empty">{emptyText}</p>
{:else}
	<ul class="rank-list">
		{#each items as item (item.value)}
			<li class="rank-item">
				<span class="rank-bar" style={`width:${(item.count / max) * 100}%`}></span>
				<span class="rank-label" title={item.value}>{item.value}</span>
				<span class="rank-count">{formatNumber(item.count)}</span>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.rank-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.rank-item {
		position: relative;
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		align-items: center;
		gap: 12px;
		padding: 7px 8px;
		border-radius: 6px;
	}

	.rank-bar {
		position: absolute;
		inset: 2px auto 2px 0;
		z-index: 0;
		border-radius: 4px;
		background: var(--accent-soft);
	}

	.rank-label,
	.rank-count {
		position: relative;
		z-index: 1;
		min-width: 0;
	}

	.rank-label {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 13px;
	}

	.rank-count {
		font-size: 12px;
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
</style>
