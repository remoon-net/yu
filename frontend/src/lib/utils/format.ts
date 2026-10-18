import { DateTime } from 'luxon';

export function formatNumber(value: number | null | undefined): string {
	return new Intl.NumberFormat('zh-CN').format(value ?? 0);
}

export function formatPercent(part: number, total: number): string {
	if (!total) return '0%';
	return `${((part / total) * 100).toFixed(1)}%`;
}

export function formatDuration(seconds: number): string {
	const total = Math.max(0, Math.round(seconds || 0));
	if (total < 60) return `${total} 秒`;

	const minutes = Math.floor(total / 60);
	const rest = total % 60;

	if (minutes < 60) {
		return rest ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`;
	}

	const hours = Math.floor(minutes / 60);
	const restMinutes = minutes % 60;
	return restMinutes ? `${hours} 小时 ${restMinutes} 分` : `${hours} 小时`;
}

export function formatDateTime(iso: string, timezone: string): string {
	const dt = DateTime.fromISO(iso, { setZone: true });
	if (!dt.isValid) return iso;
	return dt.setZone(timezone).toFormat('yyyy-LL-dd HH:mm');
}

export function formatChartTime(iso: string, timezone: string, interval: string): string {
	const dt = DateTime.fromISO(iso, { setZone: true });
	if (!dt.isValid) return iso;

	const zone = dt.setZone(timezone);
	if (interval === 'hour') return zone.toFormat('MM-dd HH:mm');
	return zone.toFormat('MM-dd');
}

export function averageDuration(durationSum: number, sessions: number): number {
	if (!sessions) return 0;
	return durationSum / sessions;
}

export function displayPath(path: string): string {
	return path || '/';
}
