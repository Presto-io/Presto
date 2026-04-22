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
  /** Content snapshot at last save point — used to detect real changes vs no-op edits */
  savedContent: '',
  /** Example content for the current template — loaded content that equals this shouldn't trigger save dialog */
  exampleContent: '',
  /** Save feedback state for the breathing light indicator */
  saveFeedback: 'idle' as 'idle' | 'saved',
});
