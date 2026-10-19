<script lang="ts">
	interface Props {
		name: string;
		size?: number;
		strokeWidth?: number;
		class?: string;
	}

	let { name, size = 16, strokeWidth = 1.8, class: className = '' }: Props = $props();

	const icons: Record<string, string[]> = {
		chart: ['M4 19V5M4 19h16M8 15l3-4 3 2 4-6'],
		pulse: ['M3 12h4l2-6 3 12 2-6h7'],
		bell: ['M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4'],
		alert: ['M12 3 2 21h20L12 3zM12 9v5M12 17h.01'],
		activity: ['M3 12h4l3-8 4 16 3-8h4'],
		link: [
			'M10 13a5 5 0 0 0 7 0l2-2a5 5 0 0 0-7-7l-1 1M14 11a5 5 0 0 0-7 0l-2 2a5 5 0 0 0 7 7l1-1'
		],
		eye: ['M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12zM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z'],
		settings: ['M4 6h16M4 12h16M4 18h16M8 4v4M16 10v4M10 16v4'],
		menu: ['M4 7h16M4 12h16M4 17h16'],
		x: ['M6 6l12 12M18 6L6 18'],
		sun: [
			'M12 17a5 5 0 1 0 0-10 5 5 0 0 0 0 10zM12 1v2M12 21v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M1 12h2M21 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4'
		],
		moon: ['M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z'],
		plus: ['M12 5v14M5 12h14'],
		trash: ['M4 7h16M10 11v6M14 11v6M6 7l1 14h10l1-14M9 7V4h6v3'],
		edit: ['M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z'],
		copy: ['M9 9h11v11H9zM5 15H4V4h11v1'],
		external: ['M14 3h7v7M21 3l-9 9M10 4H4v16h16v-6'],
		refresh: ['M20 11a8 8 0 1 0-2.3 6.7M20 5v6h-6'],
		'chevron-down': ['M6 9l6 6 6-6'],
		'chevron-left': ['M15 18l-6-6 6-6'],
		'chevron-right': ['M9 18l6-6-6-6'],
		check: ['M5 12l5 5L20 7'],
		clock: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3 2'],
		calendar: [
			'M8 3v4M16 3v4M3 9h18M5 5h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2zM8 13h.01M12 13h.01M16 13h.01M8 17h.01M12 17h.01'
		],
		qr: [
			'M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h2v2h-2zM18 14h2v2h-2zM14 18h2v2h-2zM18 18h2v2h-2z'
		],
		play: ['M8 5l12 7-12 7z'],
		mute: ['M11 5 6 9H2v6h4l5 4V5zM23 9l-6 6M17 9l6 6'],
		search: ['M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16zM21 21l-4.3-4.3'],
		user: ['M20 21a8 8 0 1 0-16 0M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z'],
		logout: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9'],
		building: [
			'M3 21h18M5 21V7l7-4 7 4v14M9 9h.01M15 9h.01M9 13h.01M15 13h.01M9 17h.01M15 17h.01'
		],
		info: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 11v5M12 8h.01'],
		arrow: ['M5 12h14M13 6l6 6-6 6'],
		filter: ['M3 5h18l-7 8v5l-4 2v-7L3 5z'],
		server: ['M4 4h16v6H4zM4 14h16v6H4zM7 7h.01M7 17h.01'],
		globe: [
			'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM3 12h18M12 3c2.5 2.5 3.5 5.5 3.5 9s-1 6.5-3.5 9c-2.5-2.5-3.5-5.5-3.5-9S9.5 5.5 12 3z'
		]
	};

	const paths = $derived(icons[name] ?? icons.info);
</script>

<svg
	class={className}
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width={strokeWidth}
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	{#each paths as d (d)}
		<path {d} />
	{/each}
</svg>
