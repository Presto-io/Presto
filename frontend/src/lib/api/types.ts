export interface Template {
  name: string;
  displayName: string;
  description: string;
  version: string;
  author: string;
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
