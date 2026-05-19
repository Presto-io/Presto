interface BatchImportResult {
	templates: { name: string; displayName: string; status: string }[];
	markdownFiles: {
		name: string;
		content: string;
		detectedTemplate?: string;
		workDir?: string;
	}[];
	workDir?: string;
}

interface WailsApp {
	GetOutputInfo: (markdown: string, templateId: string) => Promise<{
		schemaVersion: number;
		outputBaseName: string;
		previewTitle?: string;
		document?: {
			title?: string;
			authors?: string[];
			date?: string;
			keywords?: string[];
			subject?: string;
			description?: string;
			language?: string;
		};
		templateData?: Record<string, unknown>;
	}>;
	SavePDF: (markdown: string, templateId: string, workDir: string, outputBaseName: string) => Promise<void>;
	OpenFile: () => Promise<{ content: string; dir: string; path: string } | null>;
	OpenFiles: () => Promise<
		{ name: string; content: string; dir: string; isZip: boolean; path?: string }[] | null
	>;
	CompileSVG: (typstSource: string, workDir: string) => Promise<string[]>;
	PreviewUpdate: (
		markdown: string,
		templateID: string,
		workDir: string,
		documentKey: string
	) => Promise<PreviewUpdateResult>;
	PreviewStop: () => Promise<void>;
	PreviewMode: () => Promise<{
		mode: string;
		sessionId: string;
		documentVersion: number;
		retryCount: number;
		tinymistPath: string;
		fallbackMessage?: string;
	}>;
	ImportBatchZip: (filePath: string) => Promise<BatchImportResult>;
	SaveMarkdown: (content: string, filePath: string) => Promise<void>;
	SaveMarkdownAs: (content: string, defaultFilename: string) => Promise<string>;
	SaveFile: (b64Data: string, defaultFilename: string) => Promise<void>;
	GetVersion: () => Promise<string>;
	GetPlatform: () => Promise<string>;
	GetCapabilities: () => Promise<{
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
	}>;
	ShowAboutDialog: () => Promise<void>;
	IsVerbose: () => Promise<boolean>;
	CopyText: (text: string) => Promise<void>;
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
	SetDirtyState: (dirty: boolean, filename: string) => Promise<void>;
	GetStartupURL: () => Promise<string>;
	GetStartupFiles: () => Promise<
		{ name: string; content: string; dir: string; isZip: boolean; path?: string }[] | null
	>;
	SetFileOpenReady: () => Promise<void>;
	InstallTemplate: (templateName: string) => Promise<void>;
	DeleteTemplate: (name: string) => Promise<void>;
	GetInstalledTemplates: () => Promise<string[]>;
}

interface PreviewEvent {
	at: string;
	kind: string;
	seq: number;
	sessionId?: string;
	documentVersion?: number;
	mode?: string;
	dataPlaneUrl?: string;
	page?: number;
	svg?: string;
	pageHash?: string;
	error?: {
		code: string;
		message: string;
		detail?: string;
		recoverable: boolean;
	};
	diagnostics?: {
		severity: string;
		message: string;
		source?: string;
		line?: number;
		column?: number;
		mappingConfidence?: string;
	}[];
	metadata?: Record<string, unknown>;
}

interface PreviewUpdateResult {
	Version: number;
	version?: number;
	RestartSession: boolean;
	Events: PreviewEvent[];
	events?: PreviewEvent[];
	svgPages?: string[];
	SVGPages?: string[];
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
