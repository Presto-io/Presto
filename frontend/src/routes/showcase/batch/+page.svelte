<script lang="ts">
  import { ArrowLeft, Upload, FileText, X, GripVertical, Search, Package } from 'lucide-svelte';
  import { mockFiles } from '$lib/showcase/presets';

  // Mock template list for the left nav
  const mockTemplateNav = [
    { name: 'gongwen', displayName: '类公文模板' },
    { name: 'jiaoan-shicao', displayName: '实操教案模板' },
  ];

  // Build file list with IDs
  interface MockBatchFile {
    id: string;
    name: string;
    templateId: string;
    autoDetected: boolean;
    size: number;
  }

  let batchFiles: MockBatchFile[] = $state(
    mockFiles
      .filter(f => f.template)
      .map((f, i) => ({
        id: `file-${i}`,
        name: f.name,
        templateId: f.template!,
        autoDetected: f.autoDetected,
        size: Math.floor(Math.random() * 20 + 2) * 1024, // random 2-22 KB
      }))
  );

  let selectedTemplate = $state('gongwen');
  let selectedFileIds = $state<Set<string>>(new Set());

  // Group files by template
  interface TemplateGroup {
    templateId: string;
    displayName: string;
    files: MockBatchFile[];
  }

  let groups = $derived.by(() => {
    const map = new Map<string, MockBatchFile[]>();
    for (const f of batchFiles) {
      if (!map.has(f.templateId)) map.set(f.templateId, []);
      map.get(f.templateId)!.push(f);
    }
    const result: TemplateGroup[] = [];
    for (const [templateId, files] of map) {
      const tpl = mockTemplateNav.find(t => t.name === templateId);
      result.push({
        templateId,
        displayName: tpl?.displayName || templateId,
        files,
      });
    }
    return result;
  });

  function groupFileCount(templateName: string): number {
    return batchFiles.filter(f => f.templateId === templateName).length;
  }

  function toggleSelect(id: string, e: MouseEvent) {
    const newSet = new Set(selectedFileIds);
    if (e.metaKey || e.ctrlKey) {
      if (newSet.has(id)) newSet.delete(id); else newSet.add(id);
    } else if (e.shiftKey && selectedFileIds.size > 0) {
      // Range select
      const allIds = batchFiles.map(f => f.id);
      const lastSelected = [...selectedFileIds].pop()!;
      const lastIdx = allIds.indexOf(lastSelected);
      const curIdx = allIds.indexOf(id);
      const [start, end] = lastIdx < curIdx ? [lastIdx, curIdx] : [curIdx, lastIdx];
      for (let i = start; i <= end; i++) newSet.add(allIds[i]);
    } else {
      newSet.clear();
      newSet.add(id);
    }
    selectedFileIds = newSet;
  }

  // Drag and drop between groups
  let dragOverTemplate = $state<string | null>(null);

  function handleFileDragStart(e: DragEvent, fileId: string) {
    const ids = selectedFileIds.has(fileId) ? [...selectedFileIds] : [fileId];
    e.dataTransfer?.setData('application/x-presto-files', JSON.stringify(ids));
    e.dataTransfer!.effectAllowed = 'move';
  }

  function handleTemplateDragOver(e: DragEvent, templateName: string) {
    if (!e.dataTransfer?.types.includes('application/x-presto-files')) return;
    e.preventDefault();
    e.dataTransfer!.dropEffect = 'move';
    dragOverTemplate = templateName;
  }

  function handleTemplateDragLeave(templateName: string) {
    if (dragOverTemplate === templateName) dragOverTemplate = null;
  }

  function handleTemplateDrop(e: DragEvent, templateName: string) {
    e.preventDefault();
    dragOverTemplate = null;
    const data = e.dataTransfer?.getData('application/x-presto-files');
    if (!data) return;
    try {
      const ids: string[] = JSON.parse(data);
      const idSet = new Set(ids);
      batchFiles = batchFiles.map(f =>
        idSet.has(f.id) ? { ...f, templateId: templateName, autoDetected: false } : f
      );
      selectedFileIds = new Set();
    } catch { /* ignore */ }
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>批量转换</h2>
  </div>

  <div class="batch-layout">
    <!-- Left: template list -->
    <nav class="template-nav">
      <div class="nav-search">
        <Search size={14} />
        <input type="text" placeholder="搜索模板…" disabled />
      </div>
      <div class="nav-list">
        {#each mockTemplateNav as tpl (tpl.name)}
          <button
            class="nav-item"
            class:active={selectedTemplate === tpl.name}
            class:drop-target={dragOverTemplate === tpl.name}
            class:has-files={groupFileCount(tpl.name) > 0}
            onclick={() => selectedTemplate = tpl.name}
            ondragover={(e) => handleTemplateDragOver(e, tpl.name)}
            ondragleave={() => handleTemplateDragLeave(tpl.name)}
            ondrop={(e) => handleTemplateDrop(e, tpl.name)}
          >
            <span class="nav-item-name">{tpl.displayName}</span>
            {#if groupFileCount(tpl.name) > 0}
              <span class="nav-item-count">{groupFileCount(tpl.name)}</span>
            {/if}
          </button>
        {/each}
      </div>
    </nav>

    <!-- Right: batch content -->
    <div class="batch-content">
      <div class="action-bar">
        <button class="btn-action">
          <Upload size={14} />
          <span>选择文件</span>
        </button>
        <button class="btn-action subtle">
          <X size={14} />
          <span>清空</span>
        </button>
        <button class="btn-convert">
          <span>转换全部 ({batchFiles.length})</span>
        </button>
      </div>

      <div class="drop-zone compact">
        <Upload size={14} />
        <span>拖拽更多文件或 ZIP 到此处</span>
      </div>

      {#each groups as group (group.templateId)}
        <div class="section">
          <div class="section-header group-header">
            <h3>{group.displayName}</h3>
            <span class="section-count">{group.files.length}</span>
          </div>
          <div class="file-list">
            {#each group.files as bf (bf.id)}
              <div
                class="file-row batch-file-row"
                class:selected={selectedFileIds.has(bf.id)}
                onclick={(e) => toggleSelect(bf.id, e)}
                draggable="true"
                ondragstart={(e) => handleFileDragStart(e, bf.id)}
                role="option"
                aria-selected={selectedFileIds.has(bf.id)}
              >
                <span class="drag-handle"><GripVertical size={12} /></span>
                <FileText size={14} />
                <span class="file-name">{bf.name}</span>
                {#if bf.autoDetected}
                  <span class="badge-auto">自动</span>
                {/if}
                <span class="file-size">{(bf.size / 1024).toFixed(1)} KB</span>
                <button class="btn-icon" aria-label="移除">
                  <X size={12} />
                </button>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  .page {
    padding: var(--space-xl);
    padding-top: 48px;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-xl);
    flex-shrink: 0;
  }
  h2 {
    margin: 0;
    font-size: 1.125rem;
    font-family: var(--font-ui);
    color: var(--color-text-bright);
  }
  .btn-back {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    cursor: pointer;
    transition: background var(--transition);
  }
  .btn-back:hover { background: var(--color-surface-hover); }
  .batch-layout {
    display: flex;
    gap: var(--space-xl);
    flex: 1;
    min-height: 0;
  }
  .template-nav {
    width: 180px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }
  .nav-search {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    flex-shrink: 0;
  }
  .nav-search input {
    flex: 1;
    min-width: 0;
    background: none;
    border: none;
    color: var(--color-text);
    font-size: 0.75rem;
    font-family: var(--font-ui);
    outline: none;
  }
  .nav-search input::placeholder { color: var(--color-muted); }
  .nav-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow-y: auto;
    flex: 1;
  }
  .nav-item {
    text-align: left;
    padding: var(--space-sm) var(--space-md);
    background: none;
    border: 1.5px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 150ms ease;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }
  .nav-item:hover { color: var(--color-text); background: var(--color-surface); }
  .nav-item.active { color: var(--color-accent); background: var(--color-surface); }
  .nav-item.has-files { color: var(--color-text); }
  .nav-item.drop-target {
    border-color: var(--color-accent);
    background: var(--color-accent-bg-subtle);
    color: var(--color-accent);
  }
  .nav-item-name { overflow: hidden; text-overflow: ellipsis; flex: 1; }
  .nav-item-count {
    font-size: 0.625rem;
    min-width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    background: var(--color-accent);
    color: var(--color-bg);
    font-weight: 600;
    flex-shrink: 0;
    font-family: var(--font-mono);
  }
  .badge-builtin {
    font-size: 0.5625rem;
    font-weight: 600;
    padding: 0 4px;
    border-radius: 3px;
    background: var(--color-accent);
    color: var(--color-bg);
    flex-shrink: 0;
    line-height: 1.4;
  }
  .batch-content {
    flex: 1;
    min-height: 0;
    max-width: 640px;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    gap: var(--space-lg);
  }
  .action-bar {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    flex-shrink: 0;
    flex-wrap: wrap;
  }
  .btn-action {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    height: 28px;
    padding: 0 var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text);
    font-size: 0.75rem;
    cursor: pointer;
    transition: all var(--transition);
    white-space: nowrap;
  }
  .btn-action:hover { border-color: var(--color-accent); color: var(--color-accent); }
  .btn-action.subtle { color: var(--color-muted); }
  .btn-action.subtle:hover { border-color: var(--color-danger); color: var(--color-danger); }
  .btn-convert {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    height: 28px;
    padding: 0 var(--space-lg);
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
    margin-left: auto;
  }
  .btn-convert:hover { opacity: 0.85; }
  .drop-zone {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-sm);
    padding: var(--space-2xl) var(--space-xl);
    border: 1.5px dashed var(--color-border);
    border-radius: var(--radius-lg);
    color: var(--color-muted);
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .drop-zone.compact {
    flex-direction: row;
    padding: var(--space-sm) var(--space-md);
    gap: var(--space-sm);
    font-size: 0.75rem;
    border-radius: var(--radius-md);
  }
  .section {
    display: flex;
    flex-direction: column;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-sm);
  }
  .group-header {
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
  }
  h3 {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-muted);
  }
  .section-count {
    font-size: 0.6875rem;
    padding: 1px 7px;
    border-radius: 10px;
    background: var(--color-surface);
    color: var(--color-muted);
    font-family: var(--font-mono);
  }
  .file-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
  .file-row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
    color: var(--color-text);
    transition: border-color var(--transition), background var(--transition);
    cursor: default;
    user-select: none;
  }
  .file-row:hover { border-color: var(--color-surface-hover); }
  .file-row.selected {
    border-color: var(--color-accent);
    background: var(--color-accent-bg-subtle);
  }
  .drag-handle {
    color: var(--color-muted);
    cursor: grab;
    display: flex;
    align-items: center;
    opacity: 0.4;
    transition: opacity var(--transition);
    flex-shrink: 0;
  }
  .file-row:hover .drag-handle { opacity: 0.8; }
  .file-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .file-size {
    color: var(--color-muted);
    font-size: 0.6875rem;
    font-family: var(--font-mono);
    flex-shrink: 0;
  }
  .badge-auto {
    font-size: 0.5625rem;
    font-weight: 600;
    padding: 0 5px;
    border-radius: 3px;
    background: var(--color-accent-bg);
    color: var(--color-accent);
    flex-shrink: 0;
    line-height: 1.5;
  }
  .btn-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--color-muted);
    cursor: pointer;
    transition: all var(--transition);
    flex-shrink: 0;
  }
  .btn-icon:hover { color: var(--color-danger); background: var(--color-danger-bg-subtle); }
</style>
