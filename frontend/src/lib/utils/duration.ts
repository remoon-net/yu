const UNIT_SECONDS: Record<string, number> = {
	ms: 0.001,
	s: 1,
	m: 60,
	h: 3600,
	d: 86400
};

function parse(input: string, defaultUnit: 's' | 'ms'): number | null {
	const value = input.trim().toLowerCase();
	if (!value) return null;

	const match = value.match(/^(\d+(?:\.\d+)?)\s*(ms|s|m|h|d)?$/);
	if (!match) return null;

	const amount = Number(match[1]);
	const unit = match[2] ?? defaultUnit;
	const factor = UNIT_SECONDS[unit];
	if (!Number.isFinite(amount) || amount <= 0 || factor === undefined) return null;

	return amount * factor;
}

/** 解析间隔，返回秒数；不带单位时按秒处理。 */
export function parseInterval(input: string): number | null {
	const seconds = parse(input, 's');
	if (seconds === null) return null;
	return Math.round(seconds);
}

/** 解析超时，返回毫秒；不带单位时按毫秒处理。 */
export function parseTimeout(input: string): number | null {
	const seconds = parse(input, 'ms');
	if (seconds === null) return null;
	return Math.round(seconds * 1000);
}

/** 把秒数格式化为易读形式，例如 60 → 1m。 */
export function formatInterval(seconds: number): string {
	if (seconds % 86400 === 0) return `${seconds / 86400}d`;
	if (seconds % 3600 === 0) return `${seconds / 3600}h`;
	if (seconds % 60 === 0) return `${seconds / 60}m`;
	return `${seconds}s`;
}

/** 把毫秒数格式化为易读形式，例如 10000 → 10s。 */
export function formatTimeout(ms: number): string {
	if (ms >= 1000 && ms % 1000 === 0) return `${ms / 1000}s`;
	return `${ms}ms`;
}
