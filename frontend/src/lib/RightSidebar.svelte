<script lang="ts">
  import { navigate } from './router';
</script>

<div class="stack">
  <section class="card">
    <h2>What's happening</h2>
    <p class="muted">Trending hashtags will appear here.</p>
  </section>

  <section class="card">
    <h2>Subscribe to Premium</h2>
    <p class="muted">Get a blue checkmark next to your name.</p>
    <button class="btn btn-primary" type="button" onclick={() => navigate('/premium')}>
      Subscribe
    </button>
  </section>

  <section class="card">
    <h2>Who to follow</h2>
    <p class="muted">Suggested accounts will appear here.</p>
  </section>
</div>

<style lang="scss">
  .stack { display: flex; flex-direction: column; gap: 1rem; padding: 1rem; }
  .card {
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  h2 { margin: 0; font-size: 1.1rem; }
  .muted { margin: 0; font-size: 0.9rem; color: var(--text-muted); }
</style>