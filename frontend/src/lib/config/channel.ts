export type ReleaseCapabilities = {
	releaseChannel: string;
	onlineRegistry: boolean;
	onlineTemplateStore: boolean;
	onlineSkillStore: boolean;
	templateAutoUpdate: boolean;
	firstLaunchBootstrap: boolean;
	appUpdateCheck: boolean;
	externalBrowserLinks: boolean;
	localTemplateImport: boolean;
	packagedRuntimes: boolean;
};

function envChannel(): string {
	const channel = import.meta.env.VITE_PRESTO_CHANNEL ?? 'slim';
	return channel === 'portable' ? 'portable' : 'slim';
}

function capabilitiesForChannel(channel: string): ReleaseCapabilities {
	if (channel === 'portable') {
		return {
			releaseChannel: 'portable',
			onlineRegistry: false,
			onlineTemplateStore: false,
			onlineSkillStore: false,
			templateAutoUpdate: false,
			firstLaunchBootstrap: false,
			appUpdateCheck: false,
			externalBrowserLinks: false,
			localTemplateImport: true,
			packagedRuntimes: true,
		};
	}
	return {
		releaseChannel: 'slim',
		onlineRegistry: true,
		onlineTemplateStore: true,
		onlineSkillStore: true,
		templateAutoUpdate: true,
		firstLaunchBootstrap: true,
		appUpdateCheck: true,
		externalBrowserLinks: true,
		localTemplateImport: true,
		packagedRuntimes: false,
	};
}

export const defaultCapabilities = capabilitiesForChannel(envChannel());

export async function loadCapabilities(): Promise<ReleaseCapabilities> {
	try {
		const capabilities = await window.go?.main?.App?.GetCapabilities?.();
		if (capabilities) {
			return capabilities;
		}
	} catch {
		// Browser/dev fallback is driven by VITE_PRESTO_CHANNEL.
	}
	return defaultCapabilities;
}

export function isPortable(capabilities: ReleaseCapabilities): boolean {
	return capabilities.releaseChannel === 'portable';
}

export function releaseChannelLabel(capabilities: ReleaseCapabilities): string {
	return isPortable(capabilities) ? '离线便携包' : '默认精剪包';
}
