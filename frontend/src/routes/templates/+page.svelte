<script lang="ts">
  import { onMount } from 'svelte';
  import { listTemplates, discoverTemplates, installTemplate, deleteTemplate } from '$lib/api/client';
  import type { Template, GitHubRepo } from '$lib/api/types';
  import { Package, Download, Trash2, ExternalLink, Loader } from 'lucide-svelte';

  let installed: Template[] = $state([]);
  let available: GitHubRepo[] = $state([]);
  let loading = $state(true);
  let installing = $state('');

  onMount(async () => {
    try {
      const [inst, avail] = await Promise.all([listTemplates(), discoverTemplates()]);
      installed = inst ?? [];
      available = avail ?? [];
    } catch {
      // silently handle
    } finally {
      loading = false;
    }
  });

  async function handleInstall(repo: GitHubRepo) {
    installing = repo.full_name;
    try {
      await installTemplate(repo.owner.login, repo.name);
      installed = await listTemplates();
    } finally {
      installing = '';
    }
  }

  async function handleDelete(name: string) {
    if (!confirm(`确定卸载模板 "${name}"？`)) return;
    await deleteTemplate(name);
    installed = await listTemplates();
  }
</script>

<div class="page">
  <section>
    <h2>已安装模板</h2>
    {#if installed.length === 0}
      <div class="empty">
        <Package size={32} />
        <p>暂无已安装模板</p>
      </div>
    {:else}
      <div class="grid">
        {#each installed as tpl (tpl.name)}
          <div class="card">
            <div class="card-header">
              <h3>{tpl.displayName || tpl.name}</h3>
              <span class="version">v{tpl.version}</span>
            </div>
            <p class="description">{tpl.description}</p>
            <div class="card-footer">
              <span class="author">{tpl.author}</span>
              <button class="btn-danger" onclick={() => handleDelete(tpl.name)} aria-label="卸载 {tpl.name}">
                <Trash2 size={14} />
                <span>卸载</span>
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </section>

  <section>
    <h2>发现更多模板</h2>
    {#if loading}
      <div class="empty">
        <Loader size={24} />
        <p>加载中...</p>
      </div>
    {:else if available.length === 0}
      <div class="empty">
        <Package size={32} />
        <p>暂无可用模板</p>
      </div>
    {:else}
      <div class="grid">
        {#each available as repo (repo.full_name)}
          <div class="card">
            <div class="card-header">
              <h3>{repo.name}</h3>
            </div>
            <p class="description">{repo.description}</p>
            <div class="card-footer">
              <a href={repo.html_url} target="_blank" rel="noopener noreferrer" class="repo-link">
                <ExternalLink size={12} />
                <span>{repo.full_name}</span>
              </a>
              <button
                class="btn-primary"
                onclick={() => handleInstall(repo)}
                disabled={installing === repo.full_name}
              >
                {#if installing === repo.full_name}
                  <Loader size={14} />
                  <span>安装中...</span>
                {:else}
                  <Download size={14} />
                  <span>安装</span>
                {/if}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </section>
</div>

<style>
  .page {
    padding: var(--space-xl);
    max-width: 1000px;
    margin: 0 auto;
    overflow-y: auto;
    height: 100%;
  }
  section { margin-bottom: var(--space-2xl); }
  h2 {
    margin: 0 0 var(--space-lg);
    font-size: 1.25rem;
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-2xl);
    color: var(--color-muted);
  }
  .empty p { margin: 0; }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--space-md);
  }
  .card {
    background: var(--color-surface);
    padding: var(--space-lg);
    border-radius: var(--radius-lg);
    border: 1px solid var(--color-border);
    transition: border-color var(--transition);
    display: flex;
    flex-direction: column;
  }
  .card:hover { border-color: var(--color-secondary); }
  .card-header {
    display: flex;
    align-items: baseline;
    gap: var(--space-sm);
    margin-bottom: var(--space-sm);
  }
  .card-header h3 {
    margin: 0;
    font-size: 1rem;
  }
  .version {
    font-size: 0.75rem;
    color: var(--color-muted);
    font-family: var(--font-mono);
  }
  .description {
    color: var(--color-muted);
    font-size: 0.8125rem;
    margin: 0 0 var(--space-md);
    flex: 1;
    line-height: 1.5;
  }
  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .author {
    font-size: 0.75rem;
    color: var(--color-muted);
  }
  .repo-link {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: 0.75rem;
    color: var(--color-muted);
    transition: color var(--transition);
  }
  .repo-link:hover { color: var(--color-cta); }
  .btn-primary, .btn-danger {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition);
    border: none;
  }
  .btn-primary {
    background: var(--color-cta);
    color: white;
  }
  .btn-primary:hover:not(:disabled) { opacity: 0.9; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-danger {
    background: transparent;
    color: var(--color-danger);
    border: 1px solid var(--color-danger);
  }
  .btn-danger:hover {
    background: var(--color-danger);
    color: white;
  }
</style>
