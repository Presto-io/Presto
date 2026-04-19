interface WailsApp {
	SavePDF: (markdown: string, templateId: string, workDir: string) => Promise<void>;
	OpenFile: () => Promise<{ content: string; dir: string } | null>;
	OpenFiles: () => Promise<
		{ name: string; content: string; dir: string; isZip: boolean; path?: string }[] | null
	>;
	CompileSVG: (typstSource: string, workDir: string) => Promise<string[]>;
	ImportBatchZip: (filePath: string) => Promise<any>;
	SaveMarkdown: (content: string, filePath: string) => Promise<void>;
	SaveMarkdownAs: (content: string) => Promise<string>;
	SaveFile: (b64Data: string, defaultFilename: string) => Promise<void>;
	GetVersion: () => Promise<string>;
	GetPlatform: () => Promise<string>;
	SetWindowTitle: (title: string) => Promise<void>;
	ConfirmSaveDialog: (filename: string) => Promise<string>;
	QuitApp: () => Promise<void>;
	CheckAndNotifyUpdate: () => Promise<void>;
	CheckForUpdate: () => Promise<{
		hasUpdate: boolean;
		currentVersion: string;
		latestVersion: string;
		downloadURL: string;
		releaseURL: string;
	}>;
	DownloadAndInstallUpdate: (downloadURL: string) => Promise<void>;
	UpdateMenuState: (hasContent: boolean) => Promise<void>;
	GetStartupURL: () => Promise<string>;
	InstallTemplate: (templateName: string) => Promise<void>;
	DeleteTemplate: (name: string) => Promise<void>;
	GetInstalledTemplates: () => Promise<string[]>;
}

interface WailsRuntime {
	EventsOn: (eventName: string, callback: (...data: any[]) => void) => void;
	EventsOff: (eventName: string) => void;
	EventsEmit: (eventName: string, ...data: any[]) => void;
	BrowserOpenURL: (url: string) => void;
	WindowMinimise: () => void;
	WindowToggleMaximise: () => void;
	WindowSetTitle: (title: string) => void;
	Quit: () => void;
	[key: string]: any;
}

declare global {
	interface Window {
		go?: {
			main: {
				App: WailsApp;
			};
		};
		runtime?: WailsRuntime;
	}
}

export {};
