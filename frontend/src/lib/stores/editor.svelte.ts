/**
 * Module-level editor state that persists across SvelteKit navigation.
 * Transient UI state (converting, errorMsg, scroll) stays in +page.svelte.
 */
export const editor = $state({
  markdown: '',
  typstSource: '',
  svgPages: [] as string[],
  selectedTemplate: '',
  documentDir: '',
  pendingExternalLoad: false,
  currentFilePath: '',
  isDirty: false,
  documentTitle: '',
});
