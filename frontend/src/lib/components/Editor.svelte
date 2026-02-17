<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { markdown } from '@codemirror/lang-markdown';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { EditorState } from '@codemirror/state';

  let { value = $bindable(''), onchange }: {
    value?: string;
    onchange?: (val: string) => void;
  } = $props();

  let container: HTMLDivElement;
  let view: EditorView;
  let internalUpdate = false;

  onMount(() => {
    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          basicSetup,
          markdown(),
          oneDark,
          EditorView.theme({
            '&': { height: '100%', fontSize: '14px' },
            '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
            '.cm-gutters': { background: 'var(--color-background)', borderRight: '1px solid var(--color-border)' },
            '.cm-activeLineGutter': { background: 'var(--color-surface)' },
          }),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              internalUpdate = true;
              value = update.state.doc.toString();
              onchange?.(value);
              internalUpdate = false;
            }
          })
        ]
      }),
      parent: container
    });
  });

  // Sync external value changes (e.g. file upload) into CodeMirror
  $effect(() => {
    if (view && !internalUpdate) {
      const current = view.state.doc.toString();
      if (value !== current) {
        view.dispatch({
          changes: { from: 0, to: current.length, insert: value }
        });
      }
    }
  });

  onDestroy(() => view?.destroy());
</script>

<div bind:this={container} class="editor-container" role="textbox" aria-label="Markdown 编辑器"></div>

<style>
  .editor-container {
    height: 100%;
    overflow: auto;
    background: var(--color-background);
  }
  .editor-container :global(.cm-editor) {
    height: 100%;
  }
  .editor-container :global(.cm-focused) {
    outline: none;
  }
</style>
