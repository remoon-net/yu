import { api } from '#lib/api/client.js';
import type { SessionPayload, User, Workspace } from '#lib/api/types.js';

class SessionState {
	user = $state<User | null>(null);
	workspaces = $state<Workspace[]>([]);
	loaded = $state(false);
	loading = $state(false);

	async load(force = false): Promise<void> {
		if (this.loading) return;
		if (this.loaded && !force) return;

		this.loading = true;
		try {
			const data = await api.get<SessionPayload>('/api/auth/me');
			this.user = data.user;
			this.workspaces = data.workspaces;
		} catch {
			this.user = null;
			this.workspaces = [];
		} finally {
			this.loaded = true;
			this.loading = false;
		}
	}

	clear(): void {
		this.user = null;
		this.workspaces = [];
	}

	apply(data: SessionPayload): void {
		this.user = data.user;
		this.workspaces = data.workspaces;
		this.loaded = true;
	}

	async login(email: string, password: string): Promise<void> {
		const data = await api.post<SessionPayload>('/api/auth/login', { email, password });
		this.apply(data);
	}

	async logout(): Promise<void> {
		try {
			await api.post('/api/auth/logout');
		} finally {
			this.clear();
		}
	}

	findWorkspace(slug: string): Workspace | undefined {
		return this.workspaces.find((item) => item.slug === slug);
	}
}

export const session = new SessionState();

export function canManage(role: string | undefined): boolean {
	return role === 'owner' || role === 'admin';
}
