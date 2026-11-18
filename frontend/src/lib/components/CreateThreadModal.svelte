<script lang="ts">
  import type { ThreadItem } from '../api';
  import CreateThread from './CreateThread.svelte';

  let {
    isOpen = false,
    onClose,
    onCreated,
  }: {
    isOpen: boolean;
    onClose: () => void;
    onCreated?: (thread: ThreadItem) => void;
  } = $props();

  function handleBackdrop(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  function handleSuccess(thread: ThreadItem) {
    onCreated?.(thread);
    onClose();
  }
</script>

{#if isOpen}
  <div
    class="modal-backdrop"
    onclick={handleBackdrop}
    onkeydown={handleKeydown}
    tabindex="-1"
    role="dialog"
    aria-modal="true"
    aria-label="Create New Thread"
  >
    <div class="modal-box">
      <header class="modal-header">
        <button type="button" class="close-btn" onclick={onClose} aria-label="Close modal">
          ✕
        </button>
        <h2>Compose New Thread</h2>
      </header>

      <div class="modal-body">
        <CreateThread onCreated={handleSuccess} autoFocus={true} />
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(2px);
    display: grid;
    place-items: center;
    z-index: 100;
    padding: 1rem;
  }

  .modal-box {
    width: 100%;
    max-width: 600px;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 16px;
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.25);
    overflow: hidden;
    animation: scaleIn 0.15s ease-out;
  }

  @keyframes scaleIn {
    from {
      opacity: 0;
      transform: scale(0.96);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  .modal-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);

    h2 {
      margin: 0;
      font-size: 1.1rem;
      font-weight: 700;
    }
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.1rem;
    cursor: pointer;
    color: var(--text-muted);
    padding: 0.25rem 0.5rem;
    border-radius: 50%;

    &:hover {
      background: var(--border);
      color: var(--text);
    }
  }

  .modal-body {
    padding: 0;
  }
</style>
