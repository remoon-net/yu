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
	<ul class="bar-list">
		{#each items as item (item.value)}
			<li>
				<span class="bar-label" title={item.value}>{item.value}</span>
				<span class="bar-track">
					<span class="bar-fill" style="width: {(item.count / max) * 100}%"></span>
				</span>
				<span class="bar-value">{formatNumber(item.count)}</span>
			</li>
		{/each}
	</ul>
{/if}
