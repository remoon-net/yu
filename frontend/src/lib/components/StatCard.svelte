<script lang="ts">
	let {
		label,
		value,
		hint = ''
	}: {
		label: string;
		value: string | number;
		hint?: string;
	} = $props();
</script>

<div class="stat-card">
	<span class="stat-label">{label}</span>
	<strong class="stat-value">{value}</strong>
	{#if hint}
		<span class="stat-hint">{hint}</span>
	{/if}
</div>
