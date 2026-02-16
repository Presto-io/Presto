<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<a href="#main-content" class="skip-link">跳到主要内容</a>

<div class="app">
	<nav aria-label="主导航">
		<a href="/" class="logo">Presto</a>
		<div class="nav-links">
			<a href="/">编辑器</a>
			<a href="/templates">模板商店</a>
			<a href="/batch">批量转换</a>
			<a href="/settings">设置</a>
		</div>
	</nav>
	<main id="main-content">
		{@render children()}
	</main>
</div>

<style>
	.skip-link {
		position: absolute;
		top: -100%;
		left: var(--space-md);
		padding: var(--space-sm) var(--space-md);
		background: var(--color-cta);
		color: white;
		border-radius: var(--radius-md);
		z-index: 100;
		font-size: 0.875rem;
	}
	.skip-link:focus {
		top: var(--space-sm);
	}
	.app {
		display: flex;
		flex-direction: column;
		height: 100vh;
	}
	nav {
		display: flex;
		align-items: center;
		padding: var(--space-sm) var(--space-lg);
		background: var(--color-primary);
		border-bottom: 1px solid var(--color-border);
		gap: var(--space-xl);
		flex-shrink: 0;
	}
	.logo {
		font-family: var(--font-heading);
		font-weight: 700;
		font-size: 1.125rem;
		color: var(--color-cta);
		text-decoration: none;
	}
	.nav-links {
		display: flex;
		gap: var(--space-lg);
	}
	.nav-links a {
		color: var(--color-muted);
		text-decoration: none;
		font-size: 0.875rem;
		transition: color var(--transition);
	}
	.nav-links a:hover {
		color: var(--color-text);
	}
	main {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
</style>
