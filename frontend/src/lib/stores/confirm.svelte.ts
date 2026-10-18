export interface ConfirmOptions {
	title?: string;
	message: string;
	confirmText?: string;
	cancelText?: string;
	danger?: boolean;
}

interface ConfirmRequest {
	title: string;
	message: string;
	confirmText: string;
	cancelText: string;
	danger: boolean;
	resolve: (value: boolean) => void;
}

let request = $state<ConfirmRequest | null>(null);

export const confirmStore = {
	get request() {
		return request;
	}
};

/** 打开应用内确认弹窗，返回用户选择。 */
export function confirmDialog(options: ConfirmOptions): Promise<boolean> {
	request?.resolve(false);

	return new Promise((resolve) => {
		request = {
			title: options.title ?? '请确认',
			message: options.message,
			confirmText: options.confirmText ?? '确定',
			cancelText: options.cancelText ?? '取消',
			danger: options.danger ?? true,
			resolve
		};
	});
}

/** 关闭确认弹窗并回传结果。 */
export function settleConfirm(value: boolean) {
	const pending = request;
	request = null;
	pending?.resolve(value);
}
