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

export interface APIKey {
	id: string;
	workspace_id: string;
	name: string;
	prefix: string;
	created_at: string;
	last_used_at: string | null;
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
	new_visitors: number;
	returning_visitors: number;
	dau: number;
	wau: number;
	mau: number;
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
	previous: StatsOverview;
	series: TimeSeriesPoint[];
	duration_buckets: CountStat[];
	range: RangeInfo;
}

/** 维度筛选条件，空字符串表示不筛选。 */
export interface StatsFilter {
	path?: string;
	referrer_host?: string;
	country?: string;
	device?: string;
	browser?: string;
	utm_source?: string;
}

export interface RetentionCell {
	period: number;
	visitors: number;
	rate: number;
}

export interface RetentionCohort {
	cohort: string;
	size: number;
	cells: RetentionCell[];
}

export interface RetentionResponse {
	interval: 'day' | 'week';
	periods: number;
	cohorts: RetentionCohort[];
	range: RangeInfo;
}

export interface Segment {
	id: string;
	site_id: string;
	name: string;
	filters: StatsFilter;
	created_at: string;
}

export interface SegmentsResponse {
	segments: Segment[];
}

export interface SiteStatsSummary {
	site_id: string;
	name: string;
	domain: string;
	pv: number;
	uv: number;
	sessions: number;
}

export interface WorkspaceOverviewResponse {
	overview: StatsOverview;
	previous: StatsOverview;
	series: TimeSeriesPoint[];
	sites: SiteStatsSummary[];
	duration_buckets: CountStat[];
	range: RangeInfo;
}

export interface PagesResponse {
	pages: PageStat[];
	entry: CountStat[];
	exit: CountStat[];
}

export interface SourcesResponse {
	referrers: CountStat[];
	referrer_urls: CountStat[];
	channels: CountStat[];
	utm_source: CountStat[];
	utm_medium: CountStat[];
	utm_campaign: CountStat[];
	utm_term: CountStat[];
	utm_content: CountStat[];
}

export type BreakdownField =
	| 'browser'
	| 'browser_version'
	| 'os'
	| 'os_version'
	| 'device'
	| 'device_brand'
	| 'screen'
	| 'language'
	| 'country'
	| 'region'
	| 'city';

export type AudienceResponse = Record<BreakdownField, CountStat[]>;

export interface EventsResponse {
	events: EventStat[];
}

export type GoalType = 'event' | 'page';

export interface Goal {
	id: string;
	site_id: string;
	name: string;
	type: GoalType;
	match_value: string;
	created_at: string;
}

export interface GoalStat extends Goal {
	conversions: number;
	visitors: number;
	conversion_rate: number;
}

export interface GoalsResponse {
	goals: GoalStat[];
	range: RangeInfo;
}

export interface FunnelStepInput {
	type: GoalType;
	value: string;
}

export interface FunnelStepResult extends FunnelStepInput {
	visitors: number;
	conversion_rate: number;
}

export interface FunnelResponse {
	steps: FunnelStepResult[];
	range: RangeInfo;
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
	muted_until?: string | null;
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

export interface ShortLink {
	id: string;
	workspace_id: string;
	site_id: string;
	slug: string;
	target_url: string;
	title: string;
	enabled: boolean;
	expires_at: string | null;
	created_at: string;
}

export interface ShortLinkStats {
	slug: string;
	clicks: number;
	visitors: number;
	last_click: string | null;
}

export interface ShortLinksResponse {
	links: ShortLink[];
	stats: ShortLinkStats[] | null;
}

export interface ShortLinkClick {
	created_at: string;
	referrer_host: string;
	channel: string;
	utm_source: string;
	utm_campaign: string;
	browser: string;
	os: string;
	device: string;
}

export interface ShortLinkDetail {
	clicks: number;
	visitors: number;
	last_click: string | null;
	series: TimeSeriesPoint[];
	referrers: CountStat[];
	channels: CountStat[];
	utm_source: CountStat[];
	utm_campaign: CountStat[];
	browser: CountStat[];
	os: CountStat[];
	device: CountStat[];
	recent: ShortLinkClick[];
}

export interface ShortLinkStatsResponse {
	link: ShortLink;
	range: RangeInfo;
	stats: ShortLinkDetail;
}

export interface Pixel {
	id: string;
	workspace_id: string;
	site_id: string;
	slug: string;
	name: string;
	enabled: boolean;
	expires_at: string | null;
	created_at: string;
}

export interface PixelStats {
	slug: string;
	hits: number;
	visitors: number;
	last_hit: string | null;
}

export interface PixelsResponse {
	pixels: Pixel[];
	stats: PixelStats[] | null;
}

export interface PixelStatsResponse {
	pixel: Pixel;
	range: RangeInfo;
	stats: ShortLinkDetail;
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
