export interface MissingFont {
  name: string;
  displayName: string;
  url: string;
}

export interface Template {
  name: string;
  displayName: string;
  description: string;
  version: string;
  author: string;
  builtin: boolean;
  keywords?: string[];
  missingFonts?: MissingFont[];
}

export interface FieldSchema {
  type: string;
  default?: unknown;
  format?: string;
}

export interface Manifest extends Template {
  license: string;
  minPrestoVersion: string;
  frontmatterSchema?: Record<string, FieldSchema>;
}

export interface GitHubRepo {
  full_name: string;
  description: string;
  html_url: string;
  owner: { login: string };
  name: string;
}

export interface BatchFile {
  id: string;
  file: File;
  templateId: string;
  autoDetected: boolean;
  workDir?: string;
}

export interface BatchResult {
  fileId: string;
  fileName: string;
  templateId: string;
  blob?: Blob;
  error?: string;
}

export interface BatchImportResult {
  templates: { name: string; displayName: string; status: string }[];
  markdownFiles: { name: string; content: string; detectedTemplate?: string; workDir?: string }[];
  workDir?: string;
}

export interface RegistryCategory {
  id: string;
  label: { zh: string; en: string };
}

export interface PlatformInfo {
  url: string;
  sha256: string;
}

export interface RegistryTemplate {
  name: string;
  displayName: string;
  description: string;
  version: string;
  author: string;
  category: string;
  keywords: string[];
  license: string;
  trust: 'official' | 'verified' | 'community';
  publishedAt?: string;
  repository?: string;                           // v1: full URL
  repo?: string;                                 // v2: "owner/repo"
  platforms?: Record<string, PlatformInfo>;       // v2: per-platform URL + SHA256
  minPrestoVersion?: string;
  previewImage?: string;
}

export interface Registry {
  version: number;
  updatedAt: string;
  categories?: RegistryCategory[];               // v1 only; v2 derives from template.category
  templates: RegistryTemplate[];
}
