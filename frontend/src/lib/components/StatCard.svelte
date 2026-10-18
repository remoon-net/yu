<script lang="ts">
	let {
		label,
		value,
		hint = '',
		delta = null,
		invert = false
	}: {
		label: string;
		value: string | number;
		hint?: string;
		delta?: number | null;
		invert?: boolean;
	} = $props();

	const hasDelta = $derived(delta !== null && !Number.isNaN(delta));
	const tone = $derived.by(() => {
		const change = delta ?? 0;
		if (!hasDelta || change === 0) return 'flat';
		return (change > 0) !== invert ? 'good' : 'bad';
	});
</script>

<div class="stat-card">
	<span class="stat-label">{label}</span>
	<strong class="stat-value">{value}</strong>
	{#if hasDelta}
		{@const change = delta ?? 0}
		<span class="stat-delta {tone}">
			{#if Number.isFinite(change)}
				{change > 0 ? '↑' : change < 0 ? '↓' : '—'}
				{Math.abs(change).toFixed(1)}%
				<span class="stat-delta-label">较上期</span>
			{:else}
				新增
			{/if}
		</span>
	{:else if hint}
		<span class="stat-hint">{hint}</span>
	{/if}
</div>
