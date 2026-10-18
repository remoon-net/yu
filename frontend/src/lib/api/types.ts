export type Role = 'owner' | 'admin' | 'member';

export interface User {
	id: string;
	email: string;
	name: string;
	created_at: string;
}

export interface Workspace {
	id: string;
	name: string;
	slug: string;
	created_at: string;
	role: Role;
}

export interface SessionPayload {
	user: User;
	workspaces: Workspace[];
}

export interface Member {
	user_id: string;
	email: string;
	name: string;
	role: Role;
	created_at: string;
}

export interface Invite {
	id: string;
	workspace_id: string;
	email: string;
	role: Role;
	expires_at: string;
	accepted_at: string | null;
	created_at: string;
}

export interface CreateInviteResponse {
	invite_url: string;
	email: string;
	role: Role;
	expires_at: string;
	email_sent: boolean;
}

export interface Site {
	id: string;
	workspace_id: string;
	name: string;
	domain: string;
	token: string;
	share_token: string;
	timezone: string;
	created_at: string;
	script_url: string;
	endpoint: string;
	share_url: string;
	install_snippet: string;
}

export interface StatsOverview {
	pv: number;
	uv: number;
	sessions: number;
	bounces: number;
	duration_sum: number;
}

export interface TimeSeriesPoint {
	bucket: string;
	pv: number;
	uv: number;
}

export interface CountStat {
	value: string;
	count: number;
}

export interface PageStat {
	path: string;
	title: string;
	views: number;
	visitors: number;
}

export interface EventStat {
	name: string;
	count: number;
	visitors: number;
	last_seen: string;
}

export interface EventPropStat {
	key: string;
	value: string;
	count: number;
}

export interface RealtimeStats {
	visitors: number;
	sessions: number;
	pages: CountStat[];
}

export interface RangeInfo {
	from: string;
	to: string;
	interval: string;
	timezone: string;
}

export interface OverviewResponse {
	overview: StatsOverview;
	series: TimeSeriesPoint[];
	range: RangeInfo;
}

export interface PagesResponse {
	pages: PageStat[];
	entry: CountStat[];
	exit: CountStat[];
}

export interface SourcesResponse {
	referrers: CountStat[];
	channels: CountStat[];
	utm_source: CountStat[];
	utm_medium: CountStat[];
	utm_campaign: CountStat[];
}

export type BreakdownField = 'browser' | 'os' | 'device' | 'screen' | 'language' | 'country';

export type AudienceResponse = Record<BreakdownField, CountStat[]>;

export interface EventsResponse {
	events: EventStat[];
}

export interface EventDetailResponse {
	name: string;
	series: TimeSeriesPoint[];
	props: EventPropStat[];
}
