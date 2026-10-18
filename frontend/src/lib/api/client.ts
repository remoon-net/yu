export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

function readCookie(name: string): string | null {
	const prefix = name + '=';
	for (const part of document.cookie.split(';')) {
		const item = part.trim();
		if (item.startsWith(prefix)) {
			return decodeURIComponent(item.slice(prefix.length));
		}
	}
	return null;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
	const headers = new Headers(init.headers);
	const method = (init.method ?? 'GET').toUpperCase();

	if (init.body !== undefined && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	if (method !== 'GET' && method !== 'HEAD') {
		const token = readCookie('yu_csrf');
		if (token) headers.set('X-CSRF-Token', token);
	}

	const response = await fetch(path, {
		credentials: 'include',
		...init,
		headers
	});

	if (!response.ok) {
		let message = `请求失败（${response.status}）`;
		try {
			const data = (await response.json()) as { error?: string };
			if (data?.error) message = data.error;
		} catch {
			// 响应体不是 JSON 时保留默认文案
		}
		throw new ApiError(response.status, message);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	return (await response.json()) as T;
}

export const api = {
	get<T>(path: string): Promise<T> {
		return request<T>(path);
	},
	post<T>(path: string, body?: unknown): Promise<T> {
		return request<T>(path, {
			method: 'POST',
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	},
	patch<T>(path: string, body?: unknown): Promise<T> {
		return request<T>(path, {
			method: 'PATCH',
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	},
	put<T>(path: string, body?: unknown): Promise<T> {
		return request<T>(path, {
			method: 'PUT',
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	},
	delete<T>(path: string): Promise<T> {
		return request<T>(path, { method: 'DELETE' });
	}
};
