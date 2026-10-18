import { api } from '#lib/api/client.js';
import type {
	Channel,
	Incident,
	MaintenanceWindow,
	Monitor,
	MonitorCheck,
	MonitorStatsResponse,
	StatusPage,
	StatusPageItem
} from '#lib/api/types.js';

function base(workspaceId: string): string {
	return `/api/workspaces/${workspaceId}`;
}

export const monitorApi = {
	list(workspaceId: string): Promise<Monitor[]> {
		return api.get<Monitor[]>(`${base(workspaceId)}/monitors`);
	},
	get(workspaceId: string, monitorId: string): Promise<Monitor> {
		return api.get<Monitor>(`${base(workspaceId)}/monitors/${monitorId}`);
	},
	create(workspaceId: string, body: unknown): Promise<Monitor> {
		return api.post<Monitor>(`${base(workspaceId)}/monitors`, body);
	},
	update(workspaceId: string, monitorId: string, body: unknown): Promise<Monitor> {
		return api.patch<Monitor>(`${base(workspaceId)}/monitors/${monitorId}`, body);
	},
	remove(workspaceId: string, monitorId: string): Promise<void> {
		return api.delete(`${base(workspaceId)}/monitors/${monitorId}`);
	},
	stats(workspaceId: string, monitorId: string, range: string): Promise<MonitorStatsResponse> {
		return api.get<MonitorStatsResponse>(
			`${base(workspaceId)}/monitors/${monitorId}/stats?range=${encodeURIComponent(range)}`
		);
	},
	checks(workspaceId: string, monitorId: string, range = '24h'): Promise<{ checks: MonitorCheck[] }> {
		return api.get(`${base(workspaceId)}/monitors/${monitorId}/checks?range=${range}&limit=100`);
	},
	mute(workspaceId: string, monitorId: string, minutes: number): Promise<void> {
		return api.post(`${base(workspaceId)}/monitors/${monitorId}/mute`, { minutes });
	}
};

export const alertApi = {
	channels(workspaceId: string): Promise<Channel[]> {
		return api.get<Channel[]>(`${base(workspaceId)}/channels`);
	},
	createChannel(workspaceId: string, body: unknown): Promise<Channel> {
		return api.post<Channel>(`${base(workspaceId)}/channels`, body);
	},
	deleteChannel(workspaceId: string, channelId: string): Promise<void> {
		return api.delete(`${base(workspaceId)}/channels/${channelId}`);
	},
	testChannel(workspaceId: string, channelId: string): Promise<{ ok: boolean; error?: string }> {
		return api.post(`${base(workspaceId)}/channels/${channelId}/test`);
	},
	incidents(workspaceId: string): Promise<Incident[]> {
		return api.get<Incident[]>(`${base(workspaceId)}/incidents?limit=100`);
	},
	resolveIncident(workspaceId: string, incidentId: string): Promise<void> {
		return api.patch(`${base(workspaceId)}/incidents/${incidentId}/resolve`);
	},
	maintenance(workspaceId: string): Promise<MaintenanceWindow[]> {
		return api.get<MaintenanceWindow[]>(`${base(workspaceId)}/maintenance-windows`);
	},
	createMaintenance(workspaceId: string, body: unknown): Promise<MaintenanceWindow> {
		return api.post<MaintenanceWindow>(`${base(workspaceId)}/maintenance-windows`, body);
	},
	deleteMaintenance(workspaceId: string, windowId: string): Promise<void> {
		return api.delete(`${base(workspaceId)}/maintenance-windows/${windowId}`);
	}
};

export const statusApi = {
	list(workspaceId: string): Promise<StatusPage[]> {
		return api.get<StatusPage[]>(`${base(workspaceId)}/status-pages`);
	},
	create(workspaceId: string, body: unknown): Promise<StatusPage> {
		return api.post<StatusPage>(`${base(workspaceId)}/status-pages`, body);
	},
	get(workspaceId: string, pageId: string): Promise<{ page: StatusPage; items: StatusPageItem[] }> {
		return api.get(`${base(workspaceId)}/status-pages/${pageId}`);
	},
	update(workspaceId: string, pageId: string, body: unknown): Promise<StatusPage> {
		return api.patch<StatusPage>(`${base(workspaceId)}/status-pages/${pageId}`, body);
	},
	remove(workspaceId: string, pageId: string): Promise<void> {
		return api.delete(`${base(workspaceId)}/status-pages/${pageId}`);
	},
	putItems(workspaceId: string, pageId: string, items: unknown): Promise<void> {
		return api.put(`${base(workspaceId)}/status-pages/${pageId}/items`, { items });
	}
};
