import { api } from '#lib/api/client.js';
import type {
	AudienceResponse,
	EventDetailResponse,
	EventsResponse,
	OverviewResponse,
	PagesResponse,
	RealtimeStats,
	SourcesResponse
} from '#lib/api/types.js';

function base(workspaceId: string, siteId: string): string {
	return `/api/workspaces/${workspaceId}/sites/${siteId}/stats`;
}

function query(range: string): string {
	return `?range=${encodeURIComponent(range)}`;
}

export const analyticsApi = {
	overview(workspaceId: string, siteId: string, range: string): Promise<OverviewResponse> {
		return api.get<OverviewResponse>(`${base(workspaceId, siteId)}/overview${query(range)}`);
	},
	realtime(workspaceId: string, siteId: string): Promise<RealtimeStats> {
		return api.get<RealtimeStats>(`${base(workspaceId, siteId)}/realtime`);
	},
	pages(workspaceId: string, siteId: string, range: string): Promise<PagesResponse> {
		return api.get<PagesResponse>(`${base(workspaceId, siteId)}/pages${query(range)}`);
	},
	sources(workspaceId: string, siteId: string, range: string): Promise<SourcesResponse> {
		return api.get<SourcesResponse>(`${base(workspaceId, siteId)}/sources${query(range)}`);
	},
	audience(workspaceId: string, siteId: string, range: string): Promise<AudienceResponse> {
		return api.get<AudienceResponse>(`${base(workspaceId, siteId)}/audience${query(range)}`);
	},
	events(workspaceId: string, siteId: string, range: string): Promise<EventsResponse> {
		return api.get<EventsResponse>(`${base(workspaceId, siteId)}/events${query(range)}`);
	},
	eventDetail(
		workspaceId: string,
		siteId: string,
		range: string,
		name: string
	): Promise<EventDetailResponse> {
		return api.get<EventDetailResponse>(
			`${base(workspaceId, siteId)}/events/${encodeURIComponent(name)}${query(range)}`
		);
	}
};
