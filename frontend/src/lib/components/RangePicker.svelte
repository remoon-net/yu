<script lang="ts">
	let { value = $bindable('24h') }: { value: string } = $props();

	const ranges = [
		{ label: '1 小时', value: '1h' },
		{ label: '24 小时', value: '24h' },
		{ label: '7 天', value: '7d' },
		{ label: '30 天', value: '30d' }
	];
</script>

<div class="segmented">
	{#each ranges as range (range.value)}
		<button
			type="button"
			class:active={value === range.value}
			onclick={() => (value = range.value)}>{range.label}</button
		>
	{/each}
</div>
