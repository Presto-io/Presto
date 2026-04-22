<script lang="ts">
  import { onMount } from 'svelte';
  import { convertAndCompile, importBatchZip } from '$lib/api/client';
  import { createZip } from '$lib/utils/zip';
  import { templateStore } from '$lib/stores/templates.svelte';
  import { extractTemplateName, resolveTemplate } from '$lib/utils/frontmatter';
  import { pendingDrop } from '$lib/stores/pending-drop.svelte';
  import type { BatchFile, BatchResult } from '$lib/api/types';
  import { ArrowLeft, Upload, FileText, Download, X, Loader, CheckCircle, AlertCircle, Search, Package, GripVertical } from 'lucide-svelte';
  import { goto } from '$app/navigation';

  const CONCURRENCY = 3;

  let templates = $derived(templateStore.templates);
  let selectedTemplate = $state('');
  let tplSearch = $state('');
  let tplLoading = $derived(templateStore.loading);

  let filteredTemplates = $derived(
    templates.filter(tpl => {
      const q = tplSearch.toLowerCase();
      return !q ||
        (tpl.displayName || tpl.name).toLowerCase().includes(q) ||
        tpl.description.toLowerCase().includes(q);
    })
  );

  let batchFiles: BatchFile[] = $state([]);
  let results: BatchResult[] = $state([]);
  let processing = $state(false);
  let fileInput: HTMLInputElement | undefined = $state();

  // Drag-and-drop between groups
  let selectedFileIds = $state<Set<string>>(new Set());
  let dragOverTemplate = $state<string | null>(null);
  let draggingFiles = $state(false);
  let zipImporting = $state(false);

  let successCount = $derived(results.filter(r => r.blob).length);
  let errorCount = $derived(results.filter(r => r.error).length);

  // Pending = files not yet converted (no matching result)
  let convertedFileIds = $derived(new Set(results.map(r => r.fileId)));
  let pendingFiles = $derived(batchFiles.filter(f => !convertedFileIds.has(f.id)));

  // Batch progress tracking (for current conversion run)
  let batchSize = $state(0);
  let batchDone = $state(0);

  // Group files by template
  interface TemplateGroup {
    templateId: string;
    displayName: string;
    files: BatchFile[];
  }

  let groups = $derived.by(() => {
    const map = new Map<string, BatchFile[]>();
    for (const f of pendingFiles) {
      if (!map.has(f.templateId)) map.set(f.templateId, []);
      map.get(f.templateId)!.push(f);
    }
    const result: TemplateGroup[] = [];
    for (const [templateId, files] of map) {
      const tpl = templates.find(t => t.name === templateId);
      result.push({
        templateId,
        displayName: tpl?.displayName || tpl?.name || templateId,
        files,
      });
    }
    return result;
  });

  // Group results by template
  let resultGroups = $derived.by(() => {
    const map = new Map<string, BatchResult[]>();
    for (const r of results) {
      if (!map.has(r.templateId)) map.set(r.templateId, []);
      map.get(r.templateId)!.push(r);
    }
    const out: { templateId: string; displayName: string; results: BatchResult[] }[] = [];
    for (const [templateId, rs] of map) {
      const tpl = templates.find(t => t.name === templateId);
      out.push({
        templateId,
        displayName: tpl?.displayName || tpl?.name || templateId,
        results: rs,
      });
    }
    return out;
  });

  // Count files per template (for nav badge)
  function groupFileCount(templateName: string): number {
    return pendingFiles.filter(f => f.templateId === templateName).length;
  }

  // Wails desktop: SaveFile binding available?
  const wailsSaveFile: ((b64: string, name: string) => Promise<void>) | undefined =
    (window as any).go?.main?.App?.SaveFile;

  onMount(async () => {
    await templateStore.load();
    if (templates.length > 0 && !selectedTemplate) {
      selectedTemplate = templates[0].name;
    }
  });

  // Watch for files routed from layout drag-drop or page navigation
  $effect(() => {
    const data = pendingDrop.data;
    if (!data) return;
    if (!templateStore.loaded) return;

    for (const file of data.files) {
      // Per-file workDir: documentDirs (per-file) > workDir (shared fallback)
      const fileWorkDir = data.documentDirs?.get(file.name) || data.workDir;
      processDroppedFile(file, fileWorkDir);
    }
    pendingDrop.clear();
  });

  async function processDroppedFile(file: File, workDir?: string) {
    const headerText = await file.slice(0, 2048).text();
    const field = extractTemplateName(headerText);
    let templateId = selectedTemplate;
    let autoDetected = false;

    if (field) {
      const resolved = resolveTemplate(field, templates);
      if (resolved) {
        templateId = resolved;
        autoDetected = true;
      }
    }

    batchFiles = [...batchFiles, {
      id: crypto.randomUUID(),
      file,
      templateId,
      autoDetected,
      workDir,
    }];
  }

  // --- File adding with auto-detect ---
  async function addFiles(newFiles: File[], fileWorkDir?: string) {
    const additions: BatchFile[] = [];
    for (const file of newFiles) {
      // Read only the first 2KB for frontmatter detection
      const headerText = await file.slice(0, 2048).text();
      const field = extractTemplateName(headerText);
      let templateId = selectedTemplate;
      let autoDetected = false;

      if (field) {
        const resolved = resolveTemplate(field, templates);
        if (resolved) {
          templateId = resolved;
          autoDetected = true;
        }
      }

      additions.push({
        id: crypto.randomUUID(),
        file,
        templateId,
        autoDetected,
        workDir: fileWorkDir,
      });
    }
    batchFiles = [...batchFiles, ...additions];
  }

  async function handleZipImport(zipFile: File) {
    zipImporting = true;
    try {
      const result = await importBatchZip(zipFile);

      // Refresh templates if any were imported
      if (result.templates.length > 0) {
        await templateStore.refresh();
      }

      // Create BatchFile entries from markdown content
      const additions: BatchFile[] = [];
      for (const md of result.markdownFiles) {
        const blob = new Blob([md.content], { type: 'text/markdown' });
        const file = new File([blob], md.name, { type: 'text/markdown' });

        let templateId = selectedTemplate;
        let autoDetected = false;
        if (md.detectedTemplate) {
          const resolved = resolveTemplate(md.detectedTemplate, templateStore.templates);
          if (resolved) {
            templateId = resolved;
            autoDetected = true;
          }
        }

        additions.push({
          id: crypto.randomUUID(),
          file,
          templateId,
          autoDetected,
          workDir: md.workDir || result.workDir,
        });
      }
      batchFiles = [...batchFiles, ...additions];
    } catch (err) {
      console.error('ZIP import failed:', err);
    } finally {
      zipImporting = false;
    }
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const allFiles = Array.from(input.files ?? []);
    const zipFiles = allFiles.filter(f => f.name.toLowerCase().endsWith('.zip'));
    const mdFiles = allFiles.filter(f => !f.name.toLowerCase().endsWith('.zip'));
    if (mdFiles.length > 0) addFiles(mdFiles);
    for (const zip of zipFiles) handleZipImport(zip);
    input.value = '';
  }

  /** Desktop: use native dialog + Wails binding; Browser: click hidden file input */
  async function handleSelectFiles() {
    const wails = (window as any).go?.main?.App;
    if (!wails?.OpenFiles) {
      // Browser mode: use HTML file input
      fileInput?.click();
      return;
    }

    // Desktop mode: native file dialog
    const results = await wails.OpenFiles();
    if (!results || results.length === 0) return;

    for (const r of results) {
      if (r.isZip && r.path && wails.ImportBatchZip) {
        // Process ZIP via Wails binding
        zipImporting = true;
        try {
          const result = await wails.ImportBatchZip(r.path);
          if (result.templates?.length > 0) await templateStore.refresh();
          const additions: BatchFile[] = [];
          for (const md of result.markdownFiles ?? []) {
            const blob = new Blob([md.content], { type: 'text/markdown' });
            const file = new File([blob], md.name, { type: 'text/markdown' });
            let templateId = selectedTemplate;
            let autoDetected = false;
            if (md.detectedTemplate) {
              const resolved = resolveTemplate(md.detectedTemplate, templateStore.templates);
              if (resolved) {
                templateId = resolved;
                autoDetected = true;
              }
            }
            additions.push({
              id: crypto.randomUUID(),
              file,
              templateId,
              autoDetected,
              workDir: md.workDir || result.workDir,
            });
          }
          batchFiles = [...batchFiles, ...additions];
        } catch (err) {
          console.error('ImportBatchZip failed:', err);
        } finally {
          zipImporting = false;
        }
      } else if (!r.isZip) {
        const file = new File([r.content], r.name, { type: 'text/markdown' });
        addFiles([file], r.dir);
      }
    }
  }

  function removeFile(fileId: string) {
    batchFiles = batchFiles.filter(f => f.id !== fileId);
    selectedFileIds.delete(fileId);
    selectedFileIds = new Set(selectedFileIds);
  }

  function clearAll() {
    batchFiles = [];
    results = [];
    selectedFileIds = new Set();
  }

  // --- Multi-select ---
  function toggleSelect(fileId: string, event: MouseEvent) {
    if (event.metaKey || event.ctrlKey) {
      const next = new Set(selectedFileIds);
      if (next.has(fileId)) next.delete(fileId);
      else next.add(fileId);
      selectedFileIds = next;
    } else if (event.shiftKey) {
      // Range select within the same group
      const file = batchFiles.find(f => f.id === fileId);
      if (!file) return;
      const groupFiles = batchFiles.filter(f => f.templateId === file.templateId);
      const lastSelected = [...selectedFileIds].pop();
      const lastFile = lastSelected ? groupFiles.find(f => f.id === lastSelected) : null;
      if (lastFile && lastFile.templateId === file.templateId) {
        const startIdx = groupFiles.indexOf(lastFile);
        const endIdx = groupFiles.indexOf(file);
        const [from, to] = startIdx < endIdx ? [startIdx, endIdx] : [endIdx, startIdx];
        const next = new Set(selectedFileIds);
        for (let i = from; i <= to; i++) next.add(groupFiles[i].id);
        selectedFileIds = next;
      } else {
        selectedFileIds = new Set([fileId]);
      }
    } else {
      selectedFileIds = new Set([fileId]);
    }
  }

  // --- Drag-and-drop between groups ---
  function handleFileDragStart(e: DragEvent, fileId: string) {
    // Ensure the dragged file is in the selection
    if (!selectedFileIds.has(fileId)) {
      selectedFileIds = new Set([fileId]);
    }
    draggingFiles = true;
    e.dataTransfer!.effectAllowed = 'move';
    e.dataTransfer!.setData('application/x-presto-files', JSON.stringify([...selectedFileIds]));
  }

  function handleFileDragEnd() {
    draggingFiles = false;
    dragOverTemplate = null;
  }

  function handleTemplateDragOver(e: DragEvent, templateName: string) {
    // Only accept internal file drags
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
    draggingFiles = false;

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

  // Also support dropping onto group headers in the right panel
  function handleGroupHeaderDrop(e: DragEvent, templateId: string) {
    handleTemplateDrop(e, templateId);
  }

  function handleGroupHeaderDragOver(e: DragEvent) {
    if (!e.dataTransfer?.types.includes('application/x-presto-files')) return;
    e.preventDefault();
    e.dataTransfer!.dropEffect = 'move';
  }

  // --- Conversion ---
  async function convertAll() {
    if (pendingFiles.length === 0) return;
    processing = true;

    const queue = [...pendingFiles];
    batchSize = queue.length;
    batchDone = 0;

    async function worker() {
      while (queue.length > 0) {
        const bf = queue.shift()!;
        try {
          const text = await bf.file.text();
          const blob = await convertAndCompile(text, bf.templateId, bf.workDir);
          results = [...results, {
            fileId: bf.id,
            fileName: bf.file.name.replace(/\.\w+$/, '.pdf'),
            templateId: bf.templateId,
            blob,
          }];
        } catch (e) {
          results = [...results, {
            fileId: bf.id,
            fileName: bf.file.name,
            templateId: bf.templateId,
            error: e instanceof Error ? e.message : String(e),
          }];
        }
        batchDone++;
      }
    }

    await Promise.all(
      Array.from({ length: Math.min(CONCURRENCY, queue.length) }, () => worker())
    );
    processing = false;
  }

  // --- Download ---
  function blobToBase64(blob: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const dataUrl = reader.result as string;
        resolve(dataUrl.split(',')[1]);
      };
      reader.onerror = reject;
      reader.readAsDataURL(blob);
    });
  }

  async function saveViaWails(blob: Blob, filename: string) {
    const b64 = await blobToBase64(blob);
    await wailsSaveFile!(b64, filename);
  }

  function saveViaDOM(blob: Blob, filename: string) {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  async function downloadOne(r: BatchResult) {
    if (!r.blob) return;
    if (wailsSaveFile) {
      await saveViaWails(r.blob, r.fileName);
    } else {
      saveViaDOM(r.blob, r.fileName);
    }
  }

  function sanitizePath(name: string): string {
    return name.replace(/[/\\:*?"<>|]/g, '_').trim() || 'default';
  }

  async function downloadAllAsZip() {
    const successful = results.filter(r => r.blob);
    if (successful.length === 0) return;

    const zipFiles: { name: string; data: Uint8Array }[] = [];
    const nameCounters = new Map<string, number>();

    for (const r of successful) {
      const tpl = templates.find(t => t.name === r.templateId);
      const folderName = sanitizePath(tpl?.displayName || tpl?.name || r.templateId);

      // Handle duplicate filenames within same folder
      let fileName = r.fileName;
      const key = `${folderName}/${fileName}`;
      const count = nameCounters.get(key) ?? 0;
      if (count > 0) {
        const ext = fileName.lastIndexOf('.');
        fileName = ext > 0
          ? `${fileName.slice(0, ext)}-${count + 1}${fileName.slice(ext)}`
          : `${fileName}-${count + 1}`;
      }
      nameCounters.set(key, count + 1);

      const buf = await r.blob!.arrayBuffer();
      zipFiles.push({
        name: `${folderName}/${fileName}`,
        data: new Uint8Array(buf),
      });
    }

    const zipBlob = createZip(zipFiles);
    if (wailsSaveFile) {
      await saveViaWails(zipBlob, '批量转换结果.zip');
    } else {
      saveViaDOM(zipBlob, '批量转换结果.zip');
    }
  }
</script>

<div class="page">
  <div class="page-header">
    <button class="btn-back" onclick={() => goto('/')} aria-label="返回编辑器">
      <ArrowLeft size={16} />
    </button>
    <h2>批量转换</h2>
  </div>

  <div class="batch-layout">
    <!-- Left: template list (also drop targets) -->
    <nav class="template-nav">
      <div class="nav-search">
        <Search size={14} />
        <input type="text" placeholder="搜索模板…" bind:value={tplSearch} />
      </div>
      <div class="nav-list">
        {#if tplLoading}
          <div class="nav-empty"><Loader size={16} class="spin" /></div>
        {:else if filteredTemplates.length === 0}
          <div class="nav-empty">
            <Package size={20} />
            <span>{tplSearch ? '无匹配' : '暂无模板'}</span>
          </div>
        {:else}
          {#each filteredTemplates as tpl (tpl.name)}
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
              <span class="nav-item-name">{tpl.displayName || tpl.name}</span>
              {#if groupFileCount(tpl.name) > 0}
                <span class="nav-item-count">{groupFileCount(tpl.name)}</span>
              {/if}
            </button>
          {/each}
        {/if}
      </div>
    </nav>

    <!-- Right: batch content -->
    <div class="batch-content">
      <!-- Action bar -->
      <div class="action-bar">
        <button class="btn-action" onclick={handleSelectFiles}>
          <Upload size={14} />
          <span>选择文件</span>
        </button>
        <input
          bind:this={fileInput}
          type="file"
          accept=".md,.markdown,.txt,.zip"
          multiple
          onchange={handleFileInput}
          hidden
        />
        {#if batchFiles.length > 0}
          <button class="btn-action subtle" onclick={clearAll}>
            <X size={14} />
            <span>清空</span>
          </button>
          <button
            class="btn-convert"
            onclick={convertAll}
            disabled={processing || pendingFiles.length === 0}
          >
            {#if processing}
              <Loader size={14} class="spin" />
              <span>转换中 ({batchDone}/{batchSize})</span>
            {:else}
              <span>转换全部 ({pendingFiles.length})</span>
            {/if}
          </button>
        {/if}
        {#if zipImporting}
          <span class="zip-status"><Loader size={12} class="spin" /> 导入 ZIP…</span>
        {/if}
      </div>

      <!-- Drop zone / empty state -->
      {#if pendingFiles.length === 0 && results.length === 0}
        <div
          class="drop-zone"
          role="region"
          aria-label="拖拽文件区域"
        >
          <Upload size={28} strokeWidth={1.5} />
          <p class="drop-title">拖拽文件到此处</p>
          <p class="drop-hint">支持 .md .markdown .txt 文件和包含文档与模板的 .zip 包</p>
        </div>
      {:else}
        <!-- Compact drop target -->
        <div
          class="drop-zone compact"
          role="region"
          aria-label="拖拽更多文件"
        >
          <Upload size={14} />
          <span>拖拽更多文件或 ZIP 到此处</span>
        </div>
      {/if}

      <!-- File list grouped by template -->
      {#if pendingFiles.length > 0}
        {#each groups as group (group.templateId)}
          <div class="section">
            <div
              class="section-header group-header" role="listitem"
              ondragover={handleGroupHeaderDragOver}
              ondrop={(e) => handleGroupHeaderDrop(e, group.templateId)}
            >
              <h3>{group.displayName}</h3>
              <span class="section-count">{group.files.length}</span>
            </div>
            <div class="file-list">
              {#each group.files as bf (bf.id)}
                <div
                  class="file-row"
                  class:selected={selectedFileIds.has(bf.id)}
                  onclick={(e) => toggleSelect(bf.id, e)}
                  onkeydown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggleSelect(bf.id, e as any); } }}
                  draggable="true"
                  ondragstart={(e) => handleFileDragStart(e, bf.id)}
                  ondragend={handleFileDragEnd}
                  role="option" tabindex="0"
                  aria-selected={selectedFileIds.has(bf.id)}
                >
                  <span class="drag-handle"><GripVertical size={12} /></span>
                  <FileText size={14} />
                  <span class="file-name">{bf.file.name}</span>
                  {#if bf.autoDetected}
                    <span class="badge-auto">自动</span>
                  {/if}
                  <span class="file-size">{(bf.file.size / 1024).toFixed(1)} KB</span>
                  <button class="btn-icon" onclick={(e) => { e.stopPropagation(); removeFile(bf.id); }} aria-label="移除 {bf.file.name}">
                    <X size={12} />
                  </button>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      {/if}

      <!-- Results grouped by template -->
      {#if results.length > 0}
        <div class="section">
          <div class="section-header">
            <h3>转换结果</h3>
            <div class="result-summary">
              {#if successCount > 0}
                <span class="badge success"><CheckCircle size={10} /> {successCount} 成功</span>
              {/if}
              {#if errorCount > 0}
                <span class="badge error"><AlertCircle size={10} /> {errorCount} 失败</span>
              {/if}
            </div>
          </div>
          {#each resultGroups as rg (rg.templateId)}
            <div class="result-group">
              <div class="result-group-header">{rg.displayName}</div>
              <div class="file-list">
                {#each rg.results as r (r.fileId)}
                  <div class="file-row" class:has-error={!!r.error}>
                    {#if r.blob}
                      <CheckCircle size={14} />
                    {:else}
                      <AlertCircle size={14} />
                    {/if}
                    <span class="file-name">{r.fileName}</span>
                    {#if r.blob}
                      <button class="btn-dl" onclick={() => downloadOne(r)}>
                        <Download size={12} />
                        <span>下载</span>
                      </button>
                    {:else}
                      <span class="error-text" title={r.error}>{r.error}</span>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/each}
          {#if successCount > 0}
            <div class="result-actions">
              <button class="btn-zip" onclick={downloadAllAsZip}>
                <Download size={14} />
                <span>打包下载 ({successCount} 个 PDF)</span>
              </button>
            </div>
          {/if}
        </div>
      {/if}
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

  /* Two-column layout */
  .batch-layout {
    display: flex;
    gap: var(--space-xl);
    flex: 1;
    min-height: 0;
  }

  /* Left: template nav */
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
  .nav-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xl) var(--space-sm);
    color: var(--color-muted);
    font-size: 0.75rem;
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
  .nav-item:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }
  .nav-item.active {
    color: var(--color-accent);
    background: var(--color-surface);
  }
  .nav-item.has-files {
    color: var(--color-text);
  }
  .nav-item.drop-target {
    border-color: var(--color-accent);
    background: var(--color-accent-bg-subtle);
    color: var(--color-accent);
    animation: pulse-glow 0.8s ease-in-out infinite alternate;
  }
  @keyframes pulse-glow {
    from {
      border-color: var(--color-accent);
      box-shadow: 0 0 0 0 rgba(122, 162, 247, 0);
    }
    to {
      border-color: var(--color-accent);
      box-shadow: 0 0 8px 2px var(--color-accent-glow);
    }
  }
  .nav-item-name {
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }
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

  /* Right: batch content */
  .batch-content {
    flex: 1;
    min-height: 0;
    max-width: 640px;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    gap: var(--space-lg);
  }

  /* Action bar */
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
  .btn-convert:hover:not(:disabled) { opacity: 0.85; }
  .btn-convert:disabled { opacity: 0.5; cursor: not-allowed; }

  .zip-status {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: 0.6875rem;
    color: var(--color-muted);
  }

  /* Drop zone */
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
  .drop-title {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-text);
  }
  .drop-hint {
    margin: 0;
    font-size: 0.75rem;
  }

  /* Sections */
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
    transition: background var(--transition);
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
  .result-summary {
    display: flex;
    gap: var(--space-xs);
  }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.6875rem;
    padding: 1px 7px;
    border-radius: 10px;
  }
  .badge.success { background: var(--color-success-bg); color: var(--color-success); }
  .badge.error { background: var(--color-danger-bg); color: var(--color-danger); }
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

  /* Result groups */
  .result-group {
    margin-bottom: var(--space-md);
  }
  .result-group-header {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--color-muted);
    padding: var(--space-xs) 0;
    margin-bottom: var(--space-xs);
    border-bottom: 1px solid var(--color-border);
  }

  /* File list */
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
  .file-row.has-error { color: var(--color-danger); }
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

  .btn-dl {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 2px 8px;
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.6875rem;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
    flex-shrink: 0;
  }
  .btn-dl:hover { opacity: 0.85; }

  .error-text {
    font-size: 0.6875rem;
    color: var(--color-danger);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 240px;
  }

  /* Result actions */
  .result-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--space-sm);
  }
  .btn-zip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    height: 32px;
    padding: 0 var(--space-lg);
    background: var(--color-accent);
    color: var(--color-bg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: opacity var(--transition);
  }
  .btn-zip:hover { opacity: 0.85; }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
