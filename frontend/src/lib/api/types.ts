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

export interface Monitor {
	id: string;
	workspace_id: string;
	name: string;
	type: 'http' | 'tcp' | 'ping' | 'dns';
	config: Record<string, unknown>;
	interval_sec: number;
	timeout_ms: number;
	retries: number;
	alert_after: number;
	group_name: string;
	tags: string[];
	ssl_alert_days: number;
	enabled: boolean;
	next_run_at: string;
	last_status: string;
	last_checked_at: string | null;
	last_latency_ms: number | null;
	last_error: string;
	created_at: string;
}

export interface MonitorCheck {
	id: number;
	checked_at: string;
	status: string;
	latency_ms: number;
	status_code: number;
	error: string;
	cert_expires_at: string | null;
}

export interface MonitorBucket {
	bucket: string;
	total: number;
	up: number;
	p50: number;
	p95: number;
	p99: number;
}

export interface MonitorStats {
	checks: number;
	up: number;
	down: number;
	availability: number;
	p50: number;
	p95: number;
	p99: number;
	series: MonitorBucket[];
	recent: MonitorCheck[];
}

export interface MonitorStatsResponse {
	stats: MonitorStats;
	range: { from: string; to: string; interval: string };
}

export interface Channel {
	id: string;
	workspace_id: string;
	type: 'smtp' | 'webhook';
	name: string;
	config: Record<string, unknown>;
	enabled: boolean;
	created_at: string;
}

export interface Incident {
	id: string;
	monitor_id: string | null;
	monitor_name: string;
	title: string;
	body: string;
	severity: 'minor' | 'major' | 'critical';
	source: 'auto' | 'manual';
	started_at: string;
	resolved_at: string | null;
}

export interface MaintenanceWindow {
	id: string;
	title: string;
	monitor_ids: string[];
	starts_at: string;
	ends_at: string;
}

export interface StatusPage {
	id: string;
	slug: string;
	name: string;
	description: string;
	has_password: boolean;
	theme: string;
	created_at: string;
}

export interface StatusPageItem {
	monitor_id: string;
	display_name: string;
	group_name: string;
	sort_order: number;
}

export interface StatusPageItemView extends StatusPageItem {
	name: string;
	type: string;
	status: string;
	last_checked_at: string | null;
	latency_ms: number | null;
	error: string;
	uptime_24h: number;
}

export interface PublicStatusResponse {
	page: StatusPage;
	overall: string;
	items: StatusPageItemView[];
	incidents: Incident[];
	maintenance: MaintenanceWindow[];
	updated_at: string;
}

export interface ShareResponse {
	site: { name: string; domain: string; timezone: string };
	range: RangeInfo;
	overview: StatsOverview;
	series: TimeSeriesPoint[];
	pages: PageStat[];
	referrers: CountStat[];
	channels: CountStat[];
	devices: CountStat[];
}
