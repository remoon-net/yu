<script lang="ts">
	import { goto } from '$app/navigation';
	import { session } from '#lib/stores/session.svelte.js';

	let { children } = $props();

	$effect(() => {
		if (!session.loaded) {
			void session.load();
		}
	});

	$effect(() => {
		if (session.loaded && !session.user) {
			void goto('/login', { replace: true });
		}
	});
</script>

{#if session.user}
	{@render children()}
{:else}
	<div class="boot">{session.loaded ? '正在跳转…' : '加载中…'}</div>
{/if}
