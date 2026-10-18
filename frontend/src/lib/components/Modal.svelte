<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		open = false,
		title,
		children,
		onclose,
		size = 'md'
	}: {
		open: boolean;
		title: string;
		children: Snippet;
		onclose: () => void;
		size?: 'sm' | 'md' | 'lg';
	} = $props();

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') onclose();
	}
</script>

{#if open}
	<div
		class="modal-backdrop"
		role="presentation"
		onclick={onclose}
		onkeydown={handleKeydown}
	>
		<div
			class="modal"
			class:modal-sm={size === 'sm'}
			class:modal-lg={size === 'lg'}
			role="dialog"
			aria-modal="true"
			aria-label={title}
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => event.stopPropagation()}
		>
			<div class="modal-head">
				<h2>{title}</h2>
				<button class="modal-close" type="button" onclick={onclose} aria-label="关闭">×</button>
			</div>
			<div class="modal-body">
				{@render children()}
			</div>
		</div>
	</div>
{/if}
