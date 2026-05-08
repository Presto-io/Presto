const GITHUB_SEGMENT = /^[A-Za-z0-9_.-]+$/;
const PRESTO_TEMPLATE_NAME = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/;

export function githubUrlFromRepo(repo: string | undefined | null): string {
  if (!repo) return '';
  const parts = repo.split('/');
  if (parts.length !== 2 || !parts.every((part) => GITHUB_SEGMENT.test(part))) {
    return '';
  }
  return `https://github.com/${parts[0]}/${parts[1]}`;
}

export function trustedGithubUrl(raw: string | undefined | null): string {
  if (!raw) return '';
  try {
    const url = new URL(raw);
    const parts = url.pathname.split('/').filter(Boolean);
    if (
      url.protocol !== 'https:' ||
      url.hostname !== 'github.com' ||
      url.username ||
      url.password ||
      parts.length < 2 ||
      !GITHUB_SEGMENT.test(parts[0]) ||
      !GITHUB_SEGMENT.test(parts[1])
    ) {
      return '';
    }
    return url.toString();
  } catch {
    return '';
  }
}

export function isValidPrestoInstallName(name: string | undefined | null): boolean {
  return !!name && PRESTO_TEMPLATE_NAME.test(name);
}

export function isTrustedPrestoInstallUrl(raw: string | undefined | null): boolean {
  if (!raw) return false;
  try {
    const url = new URL(raw);
    return (
      url.protocol === 'presto:' &&
      url.hostname === 'install' &&
      !url.search &&
      !url.hash &&
      isValidPrestoInstallName(url.pathname.slice(1))
    );
  } catch {
    return false;
  }
}
