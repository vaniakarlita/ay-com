<script lang="ts">
  let tab = $state<'for-you' | 'following'>('for-you');
</script>

<section>
  <header class="top">
    <h1>Home</h1>
    <div class="tabs" role="tablist">
      <button type="button" role="tab" class:active={tab === 'for-you'} aria-selected={tab === 'for-you'} onclick={() => (tab = 'for-you')}>For you</button>
      <button type="button" role="tab" class:active={tab === 'following'} aria-selected={tab === 'following'} onclick={() => (tab = 'following')}>Following</button>
    </div>
  </header>

  <p class="empty">
    {tab === 'for-you' ? 'Your feed will appear here.' : 'Threads from accounts you follow will appear here.'}
  </p>
</section>

<style lang="scss">
  .top { position: sticky; top: 0; background: var(--bg); border-bottom: 1px solid var(--border); z-index: 5; }
  h1 { margin: 0; padding: 1rem 1rem 0.5rem; font-size: 1.25rem; }
  .tabs { display: flex; }
  .tabs button {
    flex: 1;
    padding: 0.9rem;
    border: 0;
    border-bottom: 3px solid transparent;
    background: none;
    color: var(--text-muted);
    font-weight: 700;
    cursor: pointer;

    &:hover { background: var(--border); }
    &.active { color: var(--text); border-bottom-color: var(--primary); }
  }
  .empty { padding: 1.5rem 1rem; color: var(--text-muted); }
</style>