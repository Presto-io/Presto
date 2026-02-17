<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { markdown } from '@codemirror/lang-markdown';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { EditorState } from '@codemirror/state';
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

  onMount(() => {
    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          basicSetup,
          markdown(),
          oneDark,
          EditorView.lineWrapping,
          zhPhrases,
          search({ top: true }),
          placeholder('在此输入 Markdown 内容，或按 ⌘O 打开文件\n\n图片语法：![描述](路径)\n  打开文件后，图片路径相对于文件所在目录\n  直接编辑时，使用绝对路径\n\n快捷键：⌘O 打开 · ⌘E 导出 · ⌘F 搜索 · ⌘, 设置'),
          EditorView.theme({
            '&': { height: '100%', fontSize: '13px' },
            '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
            '.cm-gutters': { background: 'var(--color-bg)', borderRight: '1px solid var(--color-border)' },
            '.cm-activeLineGutter': { background: 'var(--color-surface)' },
            '.cm-panels': {
              background: '#24263a',
              borderBottom: '1px solid rgba(255,255,255,0.08)',
            },
            '.cm-panels.cm-panels-top': {
              zIndex: '10',
            },
            '.cm-search': {
              padding: '6px 12px',
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              gap: '6px',
              fontSize: '13px',
              fontFamily: 'var(--font-ui)',
            },
            '.cm-search input': {
              background: '#1a1b26',
              color: '#c0caf5',
              border: '1px solid rgba(255,255,255,0.12)',
              borderRadius: '4px',
              padding: '4px 8px',
              fontSize: '13px',
              fontFamily: 'var(--font-ui)',
              outline: 'none',
              minWidth: '180px',
            },
            '.cm-search input:focus': {
              borderColor: '#7aa2f7',
              boxShadow: '0 0 0 1px #7aa2f7',
            },
            '.cm-search button': {
              background: '#2a2d44',
              color: '#c0caf5',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: '4px',
              padding: '3px 8px',
              fontSize: '12px',
              fontFamily: 'var(--font-ui)',
              cursor: 'pointer',
            },
            '.cm-search button:hover': {
              background: '#363952',
            },
            '.cm-search label': {
              fontSize: '12px',
              color: '#9aa5ce',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '4px',
              cursor: 'pointer',
            },
            '.cm-search [name=close]': {
              background: 'transparent',
              border: 'none',
              color: '#565f89',
              fontSize: '16px',
              padding: '2px 6px',
              cursor: 'pointer',
            },
            '.cm-search [name=close]:hover': {
              color: '#c0caf5',
            },
            '.cm-placeholder': {
              color: '#565f89',
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

  onDestroy(() => view?.destroy());
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
</style>
