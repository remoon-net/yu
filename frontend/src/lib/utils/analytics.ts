/** 数值型指标的环比变化百分比；上期为 0 时返回 Infinity 表示新增。 */
export function deltaOf(current: number, previous: number): number | null {
	if (!previous) return current > 0 ? Number.POSITIVE_INFINITY : null;
	return ((current - previous) / previous) * 100;
}

/** 比率类指标（跳出率、平均停留）没有上期基数时不做对比。 */
export function rateDelta(current: number, previous: number): number | null {
	if (!previous) return null;
	return ((current - previous) / previous) * 100;
}

/** 跳出率百分比。 */
export function bounceRate(bounces: number, sessions: number): number {
	return sessions ? (bounces / sessions) * 100 : 0;
}
