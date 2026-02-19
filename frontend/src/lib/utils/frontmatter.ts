/**
 * Extracts the `template` field from YAML frontmatter in markdown content.
 * Returns null if no frontmatter or no template field found.
 */
export function extractTemplateName(markdown: string): string | null {
  const trimmed = markdown.trimStart();
  if (!trimmed.startsWith('---')) return null;

  const endIdx = trimmed.indexOf('\n---', 3);
  if (endIdx === -1) return null;

  const frontmatter = trimmed.slice(3, endIdx);
  const match = frontmatter.match(/^template\s*:\s*(.+)$/m);
  if (!match) return null;

  let value = match[1].trim();
  // Strip quotes if present
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    value = value.slice(1, -1);
  }
  // Strip inline YAML comments
  const commentIdx = value.indexOf(' #');
  if (commentIdx > 0) value = value.slice(0, commentIdx).trim();

  return value || null;
}

/**
 * Resolves a template field value to an installed template's internal name.
 * Matches against `name` (exact), then `displayName` (exact), then `displayName` (case-insensitive).
 */
export function resolveTemplate(
  templateField: string,
  templates: { name: string; displayName: string }[],
): string | null {
  // Exact match on internal name
  const byName = templates.find((t) => t.name === templateField);
  if (byName) return byName.name;

  // Exact match on displayName
  const byDisplay = templates.find((t) => t.displayName === templateField);
  if (byDisplay) return byDisplay.name;

  // Case-insensitive match on displayName
  const lower = templateField.toLowerCase();
  const byDisplayCI = templates.find(
    (t) => t.displayName.toLowerCase() === lower,
  );
  if (byDisplayCI) return byDisplayCI.name;

  return null;
}
