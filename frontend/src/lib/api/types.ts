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
  publishedAt: string;
  repository: string;
}

export interface Registry {
  version: number;
  updatedAt: string;
  categories: RegistryCategory[];
  templates: RegistryTemplate[];
}
