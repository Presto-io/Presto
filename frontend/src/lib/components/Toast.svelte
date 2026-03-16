<script lang="ts">
  interface Props {
    message: string;
    type: 'success' | 'error';
    duration?: number;
    onRetry?: () => void;
  }

  let { message, type, duration = 3000, onRetry }: Props = $props();

  let visible = $state(true);

  $effect(() => {
    const timer = setTimeout(() => {
      visible = false;
    }, duration);
    return () => clearTimeout(timer);
  });
</script>

<div class="toast {type}" class:visible={visible}>
  <span class="message">{message}</span>
  {#if onRetry}
    <button class="retry-btn" onclick={() => onRetry()}>重试</button>
  {/if}
</div>

<style>
  .toast {
    position: fixed;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    padding: 12px 16px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    gap: 12px;
    z-index: 1000;
    opacity: 0;
    transition: opacity 0.3s ease;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }
  .toast.visible { opacity: 1; }
  .toast.success { background: #10b981; color: white; }
  .toast.error { background: #ef4444; color: white; }
  .message { font-size: 14px; }
  .retry-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.5);
    color: white;
    padding: 4px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 13px;
  }
  .retry-btn:hover { background: rgba(255, 255, 255, 0.1); }
</style>
