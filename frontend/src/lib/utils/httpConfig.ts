export interface HttpFormState {
	method: string;
	headers: string;
	body: string;
	expectedStatus: string;
	bodyContains: string;
	bodyNotContains: string;
	jsonPath: string;
	jsonEquals: string;
	maxLatencyMS: string;
	followRedirects: boolean;
	maxRedirects: number;
}

export const HTTP_METHODS = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'];

const HEADER_NAME = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;

export function emptyHttpForm(): HttpFormState {
	return {
		method: 'GET',
		headers: '',
		body: '',
		expectedStatus: '',
		bodyContains: '',
		bodyNotContains: '',
		jsonPath: '',
		jsonEquals: '',
		maxLatencyMS: '',
		followRedirects: true,
		maxRedirects: 5
	};
}

/** 解析「名称: 值」多行文本，空行与 # 开头的行会被忽略。 */
export function parseHeaderLines(text: string): {
	headers: Record<string, string>;
	error: string;
} {
	const headers: Record<string, string> = {};

	for (const [index, raw] of text.split('\n').entries()) {
		const line = raw.trim();
		if (!line || line.startsWith('#')) continue;

		const separator = line.indexOf(':');
		if (separator <= 0) {
			return { headers, error: `Header 第 ${index + 1} 行格式应为「名称: 值」` };
		}

		const name = line.slice(0, separator).trim();
		const value = line.slice(separator + 1).trim();

		if (!HEADER_NAME.test(name)) {
			return { headers, error: `Header 第 ${index + 1} 行名称不合法` };
		}
		if (value.length > 4096) {
			return { headers, error: `Header「${name}」的内容过长` };
		}

		headers[name] = value;
	}

	if (Object.keys(headers).length > 50) {
		return { headers, error: '自定义 Header 不能超过 50 个' };
	}

	return { headers, error: '' };
}

export function parseExpectedStatus(input: string): { value?: number[]; error: string } {
	const text = input.trim();
	if (!text) return { error: '' };

	const match = text.match(/^(\d{3})(?:\s*-\s*(\d{3}))?$/);
	if (!match) {
		return { error: '期望状态码格式应为 200 或 200-299' };
	}

	const start = Number(match[1]);
	const end = match[2] === undefined ? undefined : Number(match[2]);

	if (start < 100 || start > 599 || (end !== undefined && (end < 100 || end > 599))) {
		return { error: '期望状态码需要在 100-599 之间' };
	}
	if (end !== undefined && start > end) {
		return { error: '期望状态码起始值不能大于结束值' };
	}

	return { value: end === undefined ? [start] : [start, end], error: '' };
}

export function formatExpectedStatus(value: unknown): string {
	if (!Array.isArray(value) || value.length === 0) return '';

	const numbers = value.map(Number).filter((item) => Number.isInteger(item) && item >= 100 && item <= 599);
	if (numbers.length === 2) return `${numbers[0]}-${numbers[1]}`;
	if (numbers.length === 1) return String(numbers[0]);
	return '';
}

export function formatHeaderLines(headers: unknown): string {
	if (!headers || typeof headers !== 'object' || Array.isArray(headers)) return '';

	return Object.entries(headers as Record<string, unknown>)
		.map(([name, value]) => `${name}: ${String(value)}`)
		.join('\n');
}

function parseJSONEquals(input: string): unknown {
	const text = input.trim();
	if (!text) return undefined;

	try {
		return JSON.parse(text);
	} catch {
		// 非 JSON 文本按普通字符串处理，例如 ok。
		return text;
	}
}

/** 将表单转换为后端 http config，返回可直接提交的对象或错误。 */
export function buildHttpConfig(
	url: string,
	form: HttpFormState
): { config: Record<string, unknown>; error: string } {
	const config: Record<string, unknown> = {
		url: url.trim(),
		method: form.method.trim().toUpperCase() || 'GET'
	};

	const { headers, error: headerError } = parseHeaderLines(form.headers);
	if (headerError) return { config, error: headerError };
	if (Object.keys(headers).length > 0) config.headers = headers;

	if (form.body.trim() !== '') config.body = form.body;

	const status = parseExpectedStatus(form.expectedStatus);
	if (status.error) return { config, error: status.error };
	if (status.value) config.expected_status = status.value;

	if (form.bodyContains.trim() !== '') config.body_contains = form.bodyContains;
	if (form.bodyNotContains.trim() !== '') config.body_not_contains = form.bodyNotContains;
	if (form.jsonPath.trim() !== '') config.json_path = form.jsonPath.trim();
	if (form.jsonPath.trim() !== '') {
		const equals = parseJSONEquals(form.jsonEquals);
		if (equals !== undefined) config.json_equals = equals;
	}

	if (form.maxLatencyMS.trim() !== '') {
		const latency = Number(form.maxLatencyMS);
		if (!Number.isInteger(latency) || latency <= 0) {
			return { config, error: '耗时阈值需要是正整数（毫秒）' };
		}
		config.max_latency_ms = latency;
	}

	const maxRedirects = Number(form.maxRedirects);
	if (!Number.isInteger(maxRedirects) || maxRedirects < 1 || maxRedirects > 20) {
		return { config, error: '最大重定向次数需要在 1-20 之间' };
	}

	config.follow_redirects = form.followRedirects;
	config.max_redirects = maxRedirects;

	return { config, error: '' };
}

/** 从已有监控 config 还原表单。 */
export function httpFormFromConfig(config: Record<string, unknown>): HttpFormState {
	const jsonEquals = config.json_equals;

	return {
		method: String(config.method ?? 'GET').toUpperCase(),
		headers: formatHeaderLines(config.headers),
		body: typeof config.body === 'string' ? config.body : '',
		expectedStatus: formatExpectedStatus(config.expected_status),
		bodyContains: String(config.body_contains ?? ''),
		bodyNotContains: String(config.body_not_contains ?? ''),
		jsonPath: String(config.json_path ?? ''),
		jsonEquals:
			jsonEquals === undefined || jsonEquals === null
				? ''
				: typeof jsonEquals === 'string'
					? jsonEquals
					: JSON.stringify(jsonEquals),
		maxLatencyMS:
			typeof config.max_latency_ms === 'number' || typeof config.max_latency_ms === 'string'
				? String(config.max_latency_ms)
				: '',
		followRedirects: config.follow_redirects !== false,
		maxRedirects: Number(config.max_redirects) || 5
	};
}
