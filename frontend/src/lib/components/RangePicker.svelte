<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import {
		DEFAULT_RANGES,
		formatRangeLabel,
		makeCustomRange,
		parseCustomRange,
		toLocalInput,
		type PresetRange
	} from '#lib/utils/range.js';

	let {
		value = $bindable('24h'),
		presets = DEFAULT_RANGES,
		onchange
	}: {
		value?: string;
		presets?: PresetRange[];
		onchange?: (value: string) => void;
	} = $props();

	let open = $state(false);
	let fromInput = $state('');
	let toInput = $state('');
	let error = $state('');
	let root: HTMLDivElement | undefined = $state();

	const custom = $derived(parseCustomRange(value));
	const customLabel = $derived(custom ? formatRangeLabel(custom.from, custom.to) : '');

	function emit(next: string) {
		value = next;
		onchange?.(next);
	}

	function pick(next: string) {
		open = false;
		error = '';
		emit(next);
	}

	function toggle() {
		if (open) {
			open = false;
			return;
		}

		if (custom) {
			fromInput = toLocalInput(custom.from);
			toInput = toLocalInput(custom.to);
		} else {
			const now = new Date();
			fromInput = toLocalInput(new Date(now.getTime() - 24 * 60 * 60 * 1000));
			toInput = toLocalInput(now);
		}

		error = '';
		open = true;
	}

	function apply() {
		if (!fromInput || !toInput) {
			error = '请选择开始和结束时间';
			return;
		}

		const from = new Date(fromInput);
		const to = new Date(toInput);

		if (Number.isNaN(from.getTime()) || Number.isNaN(to.getTime())) {
			error = '时间格式不正确';
			return;
		}

		if (from >= to) {
			error = '开始时间需要早于结束时间';
			return;
		}

		pick(makeCustomRange(from.toISOString(), to.toISOString()));
	}

	function handleWindowClick(event: MouseEvent) {
		if (!open || !root) return;
		if (event.target instanceof Node && root.contains(event.target)) return;
		open = false;
	}

	function handleWindowKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') open = false;
	}
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleWindowKeydown} />

<div class="range-picker" bind:this={root}>
	<div class="segmented" role="group" aria-label="时间范围">
		{#each presets as item (item.value)}
			<button
				type="button"
				class:active={value === item.value}
				aria-pressed={value === item.value}
				onclick={() => pick(item.value)}
			>
				{item.label}
			</button>
		{/each}

		<button
			type="button"
			class="custom-btn"
			class:active={!!custom}
			aria-expanded={open}
			title={custom ? customLabel : '自定义时间范围'}
			onclick={toggle}
		>
			<Icon name="calendar" size={13} />
			<span class="custom-label">{custom ? customLabel : '自定义'}</span>
		</button>
	</div>

	{#if open}
		<div class="popover" role="dialog" aria-label="自定义时间范围">
			<header class="popover-head">
				<strong>自定义时间范围</strong>
				<span class="tiny dim">按站点时区</span>
			</header>

			<div class="popover-fields">
				<label class="field">
					开始
					<input class="input input-sm" type="datetime-local" bind:value={fromInput} />
				</label>
				<label class="field">
					结束
					<input class="input input-sm" type="datetime-local" bind:value={toInput} />
				</label>
			</div>

			{#if error}
				<p class="error-text">{error}</p>
			{/if}

			<div class="popover-actions">
				<button class="btn btn-sm btn-ghost" type="button" onclick={() => (open = false)}>
					取消
				</button>
				<button class="btn btn-sm btn-primary" type="button" onclick={apply}>应用</button>
			</div>
		</div>
	{/if}
</div>

<style>
	.range-picker {
		position: relative;
		display: inline-flex;
		max-width: 100%;
	}

	.custom-btn {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		min-width: 0;
	}

	.custom-label {
		max-width: 190px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.popover {
		position: absolute;
		right: 0;
		top: calc(100% + 8px);
		z-index: 60;
		width: min(300px, calc(100vw - 24px));
		padding: 14px;
		background: var(--panel);
		border: 1px solid var(--border);
		border-radius: 12px;
		box-shadow: var(--shadow-pop);
	}

	.popover-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 10px;
		margin-bottom: 12px;
		font-size: 13px;
	}

	.popover-fields {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 10px;
	}

	.popover-fields :global(.input) {
		width: 100%;
	}

	.popover-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		margin-top: 14px;
	}

	.error-text {
		margin-top: 10px;
		color: var(--danger);
		font-size: 12px;
	}
</style>
