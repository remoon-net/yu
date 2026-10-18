<script lang="ts">
	import '../app.css';
	import ConfirmDialog from '#lib/components/ConfirmDialog.svelte';

	let { children } = $props();
</script>

{@render children()}
<ConfirmDialog />
