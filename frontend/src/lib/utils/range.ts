export interface PresetRange {
	label: string;
	value: string;
}

export const DEFAULT_RANGES: PresetRange[] = [
	{ label: '1 小时', value: '1h' },
	{ label: '24 小时', value: '24h' },
	{ label: '7 天', value: '7d' },
	{ label: '30 天', value: '30d' }
];

const CUSTOM_PREFIX = 'custom:';

/** 自定义范围编码为 custom:<fromISO>~<toISO>，保持组件间只传 string。 */
export function isCustomRange(value: string): boolean {
	return value.startsWith(CUSTOM_PREFIX);
}

export function makeCustomRange(from: string, to: string): string {
	return `${CUSTOM_PREFIX}${from}~${to}`;
}

export function parseCustomRange(value: string): { from: string; to: string } | null {
	if (!isCustomRange(value)) return null;

	const body = value.slice(CUSTOM_PREFIX.length);
	const index = body.indexOf('~');
	if (index <= 0) return null;

	const from = body.slice(0, index);
	const to = body.slice(index + 1);
	const fromDate = new Date(from);
	const toDate = new Date(to);

	if (
		Number.isNaN(fromDate.getTime()) ||
		Number.isNaN(toDate.getTime()) ||
		fromDate >= toDate
	) {
		return null;
	}

	return { from, to };
}

/** 把 range 值转换成后端查询串：预设用 range，自定义用 from/to。 */
export function rangeQuery(value: string): string {
	const custom = parseCustomRange(value);
	if (!custom) return `?range=${encodeURIComponent(value)}`;

	const params = new URLSearchParams({ from: custom.from, to: custom.to });
	return `?${params.toString()}`;
}

/** ISO 时间 → <input type="datetime-local"> 需要的本地时间字符串。 */
export function toLocalInput(iso: string | Date): string {
	const date = iso instanceof Date ? iso : new Date(iso);
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** 自定义范围的紧凑展示，例如 10-01 09:00 ~ 10-03 18:00。 */
export function formatRangeLabel(from: string, to: string): string {
	const options: Intl.DateTimeFormatOptions = {
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit'
	};
	return `${new Date(from).toLocaleString('zh-CN', options)} ~ ${new Date(to).toLocaleString('zh-CN', options)}`;
}

