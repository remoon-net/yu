<script lang="ts">
	import { ApiError } from '#lib/api/client.js';
	import { analyticsApi } from '#lib/api/analytics.js';
	import type { GoalStat, StatsFilter } from '#lib/api/types.js';
	import { confirmDialog } from '#lib/stores/confirm.svelte.js';
	import { formatNumber } from '#lib/utils/format.js';

	let {
		workspaceId,
		siteId,
		range,
		manageable,
		filter
	}: {
		workspaceId: string;
		siteId: string;
		range: string;
		manageable: boolean;
		filter?: StatsFilter;
	} = $props();

	let goals = $state<GoalStat[]>([]);
	let loading = $state(true);
	let error = $state('');

	let showForm = $state(false);
	let editingId = $state('');
	let name = $state('');
	let type = $state<'event' | 'page'>('event');
	let matchValue = $state('');
	let saving = $state(false);
	let deletingId = $state('');
	let formError = $state('');

	$effect(() => {
		const ws = workspaceId;
		const site = siteId;
		const span = range;
		const activeFilter = filter;
		let cancelled = false;

		loading = true;
		error = '';

		analyticsApi
			.goals(ws, site, span, activeFilter)
			.then((result) => {
				if (!cancelled) goals = result.goals;
			})
			.catch((err) => {
				if (!cancelled) error = err instanceof Error ? err.message : '加载目标失败';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});

	function resetForm() {
		editingId = '';
		name = '';
		type = 'event';
		matchValue = '';
		formError = '';
	}

	function openCreate() {
		resetForm();
		showForm = true;
	}

	function openEdit(goal: GoalStat) {
		editingId = goal.id;
		name = goal.name;
		type = goal.type;
		matchValue = goal.match_value;
		formError = '';
		showForm = true;
	}

	async function load() {
		const result = await analyticsApi.goals(workspaceId, siteId, range, filter);
		goals = result.goals;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		saving = true;
		formError = '';

		try {
			const body = { name, type, match_value: matchValue };
			if (editingId) {
				await analyticsApi.updateGoal(workspaceId, siteId, editingId, body);
			} else {
				await analyticsApi.createGoal(workspaceId, siteId, body);
			}
			showForm = false;
			resetForm();
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : '保存目标失败';
		} finally {
			saving = false;
		}
	}

	async function remove(goal: GoalStat) {
		const ok = await confirmDialog({
			title: '删除目标',
			message: `确定删除目标「${goal.name}」？`,
			confirmText: '删除'
		});
		if (!ok) return;

		deletingId = goal.id;
		try {
			await analyticsApi.deleteGoal(workspaceId, siteId, goal.id);
			await load();
		} catch (err) {
			error = err instanceof ApiError ? err.message : '删除目标失败';
		} finally {
			deletingId = '';
		}
	}
</script>

<div class="card-head" style="margin-bottom: 12px">
	<div class="small muted">把事件或页面访问定义为转化目标，统计转化次数与转化率。</div>
	{#if manageable}
		<button class="btn btn-sm btn-primary" type="button" onclick={openCreate}>添加目标</button>
	{/if}
</div>

{#if error}
	<div class="error" style="margin-bottom: 12px">{error}</div>
{/if}

{#if showForm}
	<form class="card" onsubmit={submit} style="margin-bottom: 16px">
		<div class="grid-2">
			<label class="field">目标名称
				<input class="input" bind:value={name} required maxlength={80} placeholder="例如：注册" />
			</label>
			<label class="field">目标类型
				<select class="select" bind:value={type}>
					<option value="event">事件</option>
					<option value="page">页面访问</option>
				</select>
			</label>
			<label class="field">
				{type === 'event' ? '事件名' : '页面路径'}
				<input
					class="input"
					bind:value={matchValue}
					required
					maxlength={512}
					placeholder={type === 'event' ? '例如：signup' : '例如：/pricing'}
				/>
			</label>
		</div>
		{#if formError}<div class="error" style="margin-top: 12px">{formError}</div>{/if}
		<div class="modal-actions">
			<button class="btn" type="button" onclick={() => (showForm = false)}>取消</button>
			<button class="btn btn-primary" type="submit" disabled={saving}>
				{saving ? '保存中…' : editingId ? '保存' : '添加'}
			</button>
		</div>
	</form>
{/if}

{#if loading}
	<div class="card"><p class="empty">加载中…</p></div>
{:else if goals.length === 0}
	<div class="card"><p class="empty">还没有目标。添加事件或页面目标后即可查看转化数据。</p></div>
{:else}
	<div class="card" style="overflow-x: auto">
		<table class="table">
			<thead>
				<tr>
					<th>目标</th>
					<th>类型</th>
					<th>匹配</th>
					<th class="num">转化次数</th>
					<th class="num">转化访客</th>
					<th class="num">转化率</th>
					{#if manageable}<th></th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each goals as goal (goal.id)}
					<tr>
						<td>{goal.name}</td>
						<td>{goal.type === 'event' ? '事件' : '页面'}</td>
						<td class="small muted"><code>{goal.match_value}</code></td>
						<td class="num">{formatNumber(goal.conversions)}</td>
						<td class="num">{formatNumber(goal.visitors)}</td>
						<td class="num">{goal.conversion_rate.toFixed(2)}%</td>
						{#if manageable}
							<td class="num">
								<button class="btn btn-sm" type="button" onclick={() => openEdit(goal)}>编辑</button>
								<button
									class="btn btn-sm btn-danger"
									type="button"
									disabled={deletingId === goal.id}
									onclick={() => remove(goal)}
								>
									{deletingId === goal.id ? '删除中…' : '删除'}
								</button>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
