/** 按 key 记录进行中的异步操作，用于在请求期间禁用对应按钮。 */
export function createPending() {
	const active = $state<Record<string, number>>({});

	function has(key: string): boolean {
		return (active[key] ?? 0) > 0;
	}

	async function run<T>(key: string, task: () => Promise<T>): Promise<T | undefined> {
		if (has(key)) return undefined;

		active[key] = (active[key] ?? 0) + 1;
		try {
			return await task();
		} finally {
			const next = (active[key] ?? 1) - 1;
			if (next > 0) active[key] = next;
			else delete active[key];
		}
	}

	return { has, run };
}
