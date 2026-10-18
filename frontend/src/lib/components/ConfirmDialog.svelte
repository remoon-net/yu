<script lang="ts">
	import Modal from './Modal.svelte';
	import { confirmStore, settleConfirm } from '#lib/stores/confirm.svelte.js';

	const current = $derived(confirmStore.request);
</script>

{#if current}
	<Modal open={true} title={current.title} size="sm" onclose={() => settleConfirm(false)}>
		<p class="confirm-message">{current.message}</p>
		<div class="modal-actions">
			<button class="btn" type="button" onclick={() => settleConfirm(false)}>
				{current.cancelText}
			</button>
			<button
				class="btn {current.danger ? 'btn-danger' : 'btn-primary'}"
				type="button"
				onclick={() => settleConfirm(true)}
			>
				{current.confirmText}
			</button>
		</div>
	</Modal>
{/if}

<style>
	.confirm-message {
		margin: 0 0 4px;
		line-height: 1.7;
		white-space: pre-line;
	}
</style>
