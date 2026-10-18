<script lang="ts">
	import * as echarts from 'echarts/core';
	import { BarChart, LineChart } from 'echarts/charts';
	import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { EChartsCoreOption } from 'echarts/core';
	import { untrack } from 'svelte';

	echarts.use([LineChart, BarChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

	let {
		option,
		height = '280px'
	}: {
		option: EChartsCoreOption;
		height?: string;
	} = $props();

	let el: HTMLDivElement | undefined = $state();
	let chart: ReturnType<typeof echarts.init> | null = null;
	let theme = $state(document.documentElement.dataset.theme ?? 'dark');

	$effect(() => {
		const observer = new MutationObserver(() => {
			theme = document.documentElement.dataset.theme ?? 'dark';
		});

		observer.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ['data-theme']
		});

		return () => observer.disconnect();
	});

	$effect(() => {
		if (!el) return;

		const currentTheme = theme;
		chart = echarts.init(el, currentTheme === 'light' ? undefined : 'dark');
		chart.setOption(untrack(() => option), true);

		const observer = new ResizeObserver(() => chart?.resize());
		observer.observe(el);

		return () => {
			observer.disconnect();
			chart?.dispose();
			chart = null;
		};
	});

	$effect(() => {
		chart?.setOption(option, true);
	});
</script>

<div bind:this={el} style="height: {height}; width: 100%"></div>
