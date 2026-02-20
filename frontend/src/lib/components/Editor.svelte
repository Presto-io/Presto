<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { markdown } from '@codemirror/lang-markdown';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { EditorState, Compartment } from '@codemirror/state';
  import { search } from '@codemirror/search';
  import { placeholder } from '@codemirror/view';

  const zhPhrases = EditorState.phrases.of({
    "Find": "查找",
    "Replace": "替换",
    "next": "下一个",
    "previous": "上一个",
    "match case": "区分大小写",
    "regexp": "正则表达式",
    "replace": "替换",
    "replace all": "全部替换",
    "close": "关闭",
    "replaced match on line $": "已替换第 $ 行的匹配",
    "replaced $ matches": "已替换 $ 个匹配",
  });

  let { value = $bindable(''), onchange, onscroll, scrollRatio = 0 }: {
    value?: string;
    onchange?: (val: string) => void;
    onscroll?: (ratio: number) => void;
    scrollRatio?: number;
  } = $props();

  let container: HTMLDivElement;
  let view = $state<EditorView>();
  let internalUpdate = false;
  let ignoreScroll = false;

  const themeCompartment = new Compartment();
  let mqCleanup: (() => void) | undefined;

  onMount(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const isDark = mq.matches;

    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          basicSetup,
          markdown(),
          themeCompartment.of(isDark ? oneDark : []),
          EditorView.lineWrapping,
          zhPhrases,
          search({ top: true }),
          placeholder('在此输入 Markdown 内容，或按 ⌘O 打开文件\n\n图片语法：![描述](路径)\n  打开文件后，图片路径相对于文件所在目录\n  直接编辑时，使用绝对路径\n\n快捷键：⌘O 打开 · ⌘E 导出 · ⌘F 搜索 · ⌘, 设置'),
          EditorView.theme({
            '&': { height: '100%', fontSize: '13px' },
            '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
            '.cm-gutters': { background: 'var(--color-bg)', borderRight: '1px solid var(--color-border)' },
            '.cm-activeLineGutter': { background: 'var(--color-surface)' },
            '.cm-placeholder': {
              color: 'var(--color-muted)',
              fontStyle: 'italic',
              whiteSpace: 'pre-wrap',
            },
          }),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              internalUpdate = true;
              value = update.state.doc.toString();
              onchange?.(value);
              internalUpdate = false;
            }
          }),
          EditorView.domEventHandlers({
            scroll(event) {
              if (ignoreScroll) return;
              const scroller = event.target as HTMLElement;
              const maxScroll = scroller.scrollHeight - scroller.clientHeight;
              if (maxScroll > 0 && onscroll) {
                onscroll(scroller.scrollTop / maxScroll);
              }
            }
          })
        ]
      }),
      parent: container
    });

    const handleThemeChange = (e: MediaQueryListEvent) => {
      view?.dispatch({
        effects: themeCompartment.reconfigure(e.matches ? oneDark : [])
      });
    };
    mq.addEventListener('change', handleThemeChange);
    mqCleanup = () => mq.removeEventListener('change', handleThemeChange);
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

  // Sync scroll from preview
  $effect(() => {
    if (view && scrollRatio >= 0 && !ignoreScroll) {
      const scroller = view.scrollDOM;
      const maxScroll = scroller.scrollHeight - scroller.clientHeight;
      if (maxScroll > 0) {
        ignoreScroll = true;
        scroller.scrollTop = scrollRatio * maxScroll;
        requestAnimationFrame(() => { ignoreScroll = false; });
      }
    }
  });

  onDestroy(() => { mqCleanup?.(); view?.destroy(); });
</script>

<div bind:this={container} class="editor-container" role="textbox" aria-label="Markdown 编辑器"></div>

<style>
  .editor-container {
    height: 100%;
    overflow: auto;
    background: var(--color-bg);
  }
  .editor-container :global(.cm-editor) {
    height: 100%;
  }
  .editor-container :global(.cm-focused) {
    outline: none;
  }

  /* ── Search Panel: VSCode-style ────────────────────────── */

  .editor-container :global(.cm-panels) {
    background: var(--color-panel-bg);
    border-bottom: 1px solid var(--color-border);
  }
  .editor-container :global(.cm-panels.cm-panels-top) {
    z-index: 10;
  }

  /* Panel layout */
  .editor-container :global(.cm-search) {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 2px;
    padding: 6px 30px 6px 8px;
    font-size: 13px;
    font-family: var(--font-ui);
    position: relative;
  }

  /* <br> as flex row break */
  .editor-container :global(.cm-search br) {
    flex-basis: 100%;
    height: 0;
    margin: 1px 0;
    border: none;
  }

  /* Reorder: input → toggles → nav buttons (like VSCode) */
  .editor-container :global(.cm-search input[name=search]) { order: 0; }
  .editor-container :global(.cm-search label:has(input[name=case])) { order: 1; }
  .editor-container :global(.cm-search label:has(input[name=re])) { order: 2; }
  .editor-container :global(.cm-search label:has(input[name=word])) { order: 3; }
  .editor-container :global(.cm-search button[name=prev]) { order: 4; margin-left: 4px; }
  .editor-container :global(.cm-search button[name=next]) { order: 5; }
  .editor-container :global(.cm-search button[name=select]) { order: 6; display: none; }
  .editor-container :global(.cm-search br) { order: 7; }
  .editor-container :global(.cm-search input[name=replace]) { order: 8; }
  .editor-container :global(.cm-search button[name=replace]) { order: 9; }
  .editor-container :global(.cm-search button[name=replaceAll]) { order: 10; }

  /* Text inputs */
  .editor-container :global(.cm-search .cm-textfield) {
    flex: 1 1 180px;
    background: var(--color-bg);
    color: var(--color-text);
    border: 1px solid var(--color-border-input);
    border-radius: 3px;
    padding: 3px 8px;
    font-size: 13px;
    font-family: var(--font-mono);
    outline: none;
    height: 26px;
    box-sizing: border-box;
    margin: 0;
  }
  .editor-container :global(.cm-search .cm-textfield:focus) {
    border-color: var(--color-accent);
  }

  /* ── Icon Buttons ──────────────────────────────────────── */

  .editor-container :global(.cm-search .cm-button) {
    background: transparent;
    color: var(--color-muted);
    border: 1px solid transparent;
    border-radius: 3px;
    padding: 0;
    font-size: 0;
    cursor: pointer;
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin: 0;
    transition: background 0.1s, color 0.1s;
  }
  .editor-container :global(.cm-search .cm-button:hover) {
    background: var(--color-hover-overlay);
    color: var(--color-text);
  }

  /* Button icons via ::after */
  .editor-container :global(.cm-search button[name=prev]::after) {
    content: '↑';
    font-size: 16px;
  }
  .editor-container :global(.cm-search button[name=next]::after) {
    content: '↓';
    font-size: 16px;
  }
  .editor-container :global(.cm-search button[name=replace]::after) {
    content: '↦';
    font-size: 16px;
  }
  .editor-container :global(.cm-search button[name=replaceAll]::after) {
    content: '⇉';
    font-size: 16px;
  }

  /* ── Toggle Labels (Aa, .*, ab) ────────────────────────── */

  .editor-container :global(.cm-search label) {
    font-size: 0;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 24px;
    border-radius: 3px;
    border: 1px solid transparent;
    margin: 0;
    flex-shrink: 0;
    color: var(--color-muted);
    transition: background 0.1s, color 0.1s, border-color 0.1s;
  }
  .editor-container :global(.cm-search label:hover) {
    background: var(--color-hover-overlay);
    color: var(--color-text);
  }

  /* Hide checkbox inputs */
  .editor-container :global(.cm-search input[type=checkbox]) {
    display: none;
  }

  /* Toggle icons */
  .editor-container :global(.cm-search label:has(input[name=case])::after) {
    content: 'Aa';
    font-size: 13px;
    font-weight: 600;
  }
  .editor-container :global(.cm-search label:has(input[name=re])::after) {
    content: '.*';
    font-size: 13px;
    font-weight: 600;
  }
  .editor-container :global(.cm-search label:has(input[name=word])::after) {
    content: 'ab';
    font-size: 12px;
    font-weight: 600;
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  /* Active toggle state */
  .editor-container :global(.cm-search label:has(input[type=checkbox]:checked)) {
    background: var(--color-accent-bg);
    border-color: var(--color-accent-border);
    color: var(--color-accent);
  }

  /* ── Close Button ──────────────────────────────────────── */

  .editor-container :global(.cm-search [name=close]) {
    position: absolute;
    top: 6px;
    right: 6px;
    background: transparent;
    border: none;
    color: var(--color-muted);
    font-size: 0;
    padding: 0;
    cursor: pointer;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
    transition: background 0.1s, color 0.1s;
  }
  .editor-container :global(.cm-search [name=close]:hover) {
    color: var(--color-text);
    background: var(--color-hover-overlay);
  }
  .editor-container :global(.cm-search [name=close]::after) {
    content: '×';
    font-size: 16px;
  }

  /* ── Search Match Highlights ───────────────────────────── */

  .editor-container :global(.cm-searchMatch) {
    background-color: var(--color-search-match);
    outline: 1px solid var(--color-search-match-border);
    border-radius: 2px;
  }
  .editor-container :global(.cm-searchMatch-selected) {
    background-color: var(--color-search-match-active);
    outline: 1px solid var(--color-search-match-active-border);
  }
</style>
