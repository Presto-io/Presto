<script lang="ts">
	import { flip } from 'svelte/animate';
	import { cubicOut } from 'svelte/easing';
	import { fade, fly } from 'svelte/transition';
	import { AlertTriangle, Check, CircleAlert, Info, X } from 'lucide-svelte';
	import {
		notificationStore,
		type NotificationItem,
		type NotificationType,
	} from '$lib/stores/notification.svelte';

	function iconFor(type: NotificationType) {
		switch (type) {
			case 'success':
				return Check;
			case 'warning':
				return AlertTriangle;
			case 'error':
				return CircleAlert;
			default:
				return Info;
		}
	}

	function transitionDelay(index: number) {
		return Math.min(index * 28, 84);
	}

	function stackStyle(index: number) {
		return `--stack-index: ${index}; --stack-shift: ${Math.min(index * 7, 14)}px;`;
	}

	function actionTone(item: NotificationItem) {
		return item.type === 'error' ? 'danger' : 'accent';
	}
</script>

{#if notificationStore.visibleItems.length > 0}
	<div class="notification-center" aria-live="polite" aria-atomic="false">
		{#each notificationStore.visibleItems as notification, index (notification.id)}
			{@const Icon = iconFor(notification.type)}
			<article
				class="notification-shell"
				animate:flip={{ duration: 260, easing: cubicOut }}
				in:fly={{ x: 30, y: -8, duration: 260, opacity: 0.2, delay: transitionDelay(index) }}
				out:fade={{ duration: 220 }}
				style={stackStyle(index)}
			>
				<div class="notification-card tone-{notification.type}">
					<div class="notification-icon">
						<Icon size={15} strokeWidth={2.2} />
					</div>

					<div class="notification-body">
						<p class="notification-message">{notification.message}</p>
						{#if notification.detail}
							<p class="notification-detail">{notification.detail}</p>
						{/if}

						{#if notification.action}
							<button
								type="button"
								class="notification-action {actionTone(notification)}"
								onclick={() => notificationStore.runAction(notification.id)}
							>
								{notification.action.label}
							</button>
						{/if}
					</div>

					<button
						type="button"
						class="notification-close"
						aria-label="关闭通知"
						onclick={() => notificationStore.dismiss(notification.id)}
					>
						<X size={14} />
					</button>
				</div>
			</article>
		{/each}
	</div>
{/if}

<style>
	.notification-center {
		position: fixed;
		top: var(--space-xl);
		right: var(--space-xl);
		z-index: 9200;
		display: flex;
		flex-direction: column;
		align-items: stretch;
		gap: 12px;
		width: min(360px, calc(100vw - 32px));
		pointer-events: none;
	}

	.notification-shell {
		pointer-events: auto;
	}

	.notification-card {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 12px;
		align-items: start;
		padding: 14px 14px 13px;
		border-radius: 16px;
		border: 1px solid var(--color-border);
		background:
			linear-gradient(145deg, color-mix(in srgb, var(--color-surface) 88%, white 12%), var(--color-surface)),
			var(--color-surface);
		box-shadow:
			0 18px 38px rgba(0, 0, 0, 0.24),
			0 1px 0 rgba(255, 255, 255, 0.05) inset;
		backdrop-filter: blur(18px);
		transform:
			translateX(calc(var(--stack-shift) * -1))
			scale(calc(1 - (var(--stack-index) * 0.025)))
			rotate(calc(var(--stack-index) * -0.8deg));
		transform-origin: top right;
	}

	.notification-card.tone-success {
		border-color: color-mix(in srgb, var(--color-success) 55%, var(--color-border));
	}

	.notification-card.tone-info {
		border-color: color-mix(in srgb, var(--color-accent) 45%, var(--color-border));
	}

	.notification-card.tone-warning {
		border-color: color-mix(in srgb, var(--color-warning) 55%, var(--color-border));
	}

	.notification-card.tone-error {
		border-color: color-mix(in srgb, var(--color-danger) 58%, var(--color-border));
	}

	.notification-icon {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		border-radius: 999px;
		margin-top: 1px;
		background: var(--color-accent-bg-subtle);
		color: var(--color-accent);
		flex-shrink: 0;
	}

	.notification-card.tone-success .notification-icon {
		background: var(--color-success-bg);
		color: var(--color-success);
	}

	.notification-card.tone-warning .notification-icon {
		background: color-mix(in srgb, var(--color-warning) 18%, transparent);
		color: var(--color-warning);
	}

	.notification-card.tone-error .notification-icon {
		background: var(--color-danger-bg);
		color: var(--color-danger);
	}

	.notification-body {
		display: flex;
		flex-direction: column;
		gap: 6px;
		min-width: 0;
	}

	.notification-message {
		margin: 0;
		color: var(--color-text-bright);
		font-size: 0.9rem;
		font-weight: 600;
		line-height: 1.45;
		word-break: break-word;
	}

	.notification-detail {
		margin: 0;
		color: var(--color-muted);
		font-size: 0.77rem;
		line-height: 1.5;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.notification-action {
		align-self: flex-start;
		margin-top: 2px;
		padding: 6px 10px;
		border-radius: 999px;
		font-size: 0.76rem;
		font-weight: 600;
		background: var(--color-accent-bg);
		color: var(--color-accent);
	}

	.notification-action.accent:hover {
		background: color-mix(in srgb, var(--color-accent-bg) 78%, transparent);
	}

	.notification-action.danger {
		background: var(--color-danger-bg);
		color: var(--color-danger);
	}

	.notification-action.danger:hover {
		background: color-mix(in srgb, var(--color-danger-bg) 82%, transparent);
	}

	.notification-close {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		border-radius: 999px;
		background: transparent;
		color: var(--color-muted);
	}

	.notification-close:hover {
		background: var(--color-hover-overlay);
		color: var(--color-text-bright);
	}

	@media (max-width: 640px) {
		.notification-center {
			top: 14px;
			right: 14px;
			left: 14px;
			width: auto;
		}
	}
</style>
