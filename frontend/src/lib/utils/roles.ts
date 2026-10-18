export const roleLabels: Record<string, string> = {
	owner: '所有者',
	admin: '管理员',
	member: '成员'
};

export function roleLabel(role: string | undefined): string {
	return role ? (roleLabels[role] ?? role) : '成员';
}
