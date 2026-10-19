import { api } from '#lib/api/client.js';
import type {
	AudienceResponse,
	EventDetailResponse,
	EventsResponse,
	FunnelResponse,
	FunnelStepInput,
	Goal,
	GoalsResponse,
	OverviewResponse,
	PagesResponse,
	RealtimeStats,
	RetentionResponse,
	Segment,
	SegmentsResponse,
	SourcesResponse,
	StatsFilter,
	WorkspaceOverviewResponse
} from '#lib/api/types.js';
import { rangeQuery } from '#lib/utils/range.js';

function base(workspaceId: string, siteId: string): string {
	return `/api/workspaces/${workspaceId}/sites/${siteId}/stats`;
}

function query(range: string): string {
	return rangeQuery(range);
}

/** 拼接到已有 query 后面的筛选参数，忽略空值。 */
function filterQuery(filter?: StatsFilter): string {
	if (!filter) return '';

	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(filter)) {
		const trimmed = value?.trim();
		if (trimmed) params.set(key, trimmed);
	}

	const encoded = params.toString();
	return encoded ? `&${encoded}` : '';
}

export const analyticsApi = {
	overview(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<OverviewResponse> {
		return api.get<OverviewResponse>(
			`${base(workspaceId, siteId)}/overview${query(range)}${filterQuery(filter)}`
		);
	},
	retention(
		workspaceId: string,
		siteId: string,
		range: string,
		interval: 'day' | 'week',
		filter?: StatsFilter
	): Promise<RetentionResponse> {
		return api.get<RetentionResponse>(
			`${base(workspaceId, siteId)}/retention${query(range)}&interval=${interval}${filterQuery(filter)}`
		);
	},
	segments(workspaceId: string, siteId: string): Promise<SegmentsResponse> {
		return api.get<SegmentsResponse>(`/api/workspaces/${workspaceId}/sites/${siteId}/segments`);
	},
	createSegment(
		workspaceId: string,
		siteId: string,
		body: { name: string; filters: StatsFilter }
	): Promise<Segment> {
		return api.post<Segment>(`/api/workspaces/${workspaceId}/sites/${siteId}/segments`, body);
	},
	updateSegment(
		workspaceId: string,
		siteId: string,
		segmentId: string,
		body: { name: string; filters: StatsFilter }
	): Promise<Segment> {
		return api.patch<Segment>(
			`/api/workspaces/${workspaceId}/sites/${siteId}/segments/${segmentId}`,
			body
		);
	},
	deleteSegment(workspaceId: string, siteId: string, segmentId: string): Promise<void> {
		return api.delete(
			`/api/workspaces/${workspaceId}/sites/${siteId}/segments/${segmentId}`
		);
	},
	workspaceOverview(
		workspaceId: string,
		range: string,
		timezone: string
	): Promise<WorkspaceOverviewResponse> {
		const params = rangeQuery(range);
		return api.get<WorkspaceOverviewResponse>(
			`/api/workspaces/${workspaceId}/stats/overview${params}&timezone=${encodeURIComponent(timezone)}`
		);
	},
	realtime(workspaceId: string, siteId: string): Promise<RealtimeStats> {
		return api.get<RealtimeStats>(`${base(workspaceId, siteId)}/realtime`);
	},
	pages(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<PagesResponse> {
		return api.get<PagesResponse>(
			`${base(workspaceId, siteId)}/pages${query(range)}${filterQuery(filter)}`
		);
	},
	sources(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<SourcesResponse> {
		return api.get<SourcesResponse>(
			`${base(workspaceId, siteId)}/sources${query(range)}${filterQuery(filter)}`
		);
	},
	audience(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<AudienceResponse> {
		return api.get<AudienceResponse>(
			`${base(workspaceId, siteId)}/audience${query(range)}${filterQuery(filter)}`
		);
	},
	events(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<EventsResponse> {
		return api.get<EventsResponse>(
			`${base(workspaceId, siteId)}/events${query(range)}${filterQuery(filter)}`
		);
	},
	eventDetail(
		workspaceId: string,
		siteId: string,
		range: string,
		name: string,
		filter?: StatsFilter
	): Promise<EventDetailResponse> {
		return api.get<EventDetailResponse>(
			`${base(workspaceId, siteId)}/events/${encodeURIComponent(name)}${query(range)}${filterQuery(filter)}`
		);
	},
	goals(
		workspaceId: string,
		siteId: string,
		range: string,
		filter?: StatsFilter
	): Promise<GoalsResponse> {
		return api.get<GoalsResponse>(
			`${base(workspaceId, siteId)}/goals/stats${query(range)}${filterQuery(filter)}`
		);
	},
	createGoal(
		workspaceId: string,
		siteId: string,
		body: { name: string; type: string; match_value: string }
	): Promise<Goal> {
		return api.post<Goal>(`${base(workspaceId, siteId)}/goals`, body);
	},
	updateGoal(
		workspaceId: string,
		siteId: string,
		goalId: string,
		body: { name: string; type: string; match_value: string }
	): Promise<Goal> {
		return api.patch<Goal>(`${base(workspaceId, siteId)}/goals/${goalId}`, body);
	},
	deleteGoal(workspaceId: string, siteId: string, goalId: string): Promise<void> {
		return api.delete(`${base(workspaceId, siteId)}/goals/${goalId}`);
	},
	funnel(
		workspaceId: string,
		siteId: string,
		range: string,
		steps: FunnelStepInput[],
		filter?: StatsFilter
	): Promise<FunnelResponse> {
		return api.post<FunnelResponse>(
			`${base(workspaceId, siteId)}/funnel${query(range)}${filterQuery(filter)}`,
			{ steps }
		);
	},
	exportUrl(workspaceId: string, siteId: string, range: string): string {
		return `${base(workspaceId, siteId)}/export${query(range)}`;
	}
};
