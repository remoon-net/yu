/**
 * 把「逗号 / 空白分隔」的输入解析成去重后的标签数组。
 * 与后端校验保持一致：最多 20 个、忽略空项。
 */
export function parseTags(raw: string): string[] {
	return [
		...new Set(
			raw
				.split(/[,，\s]+/)
				.map((item) => item.trim())
				.filter(Boolean)
		)
	].slice(0, 20);
}
