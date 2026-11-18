<script lang="ts">
  let { title }: { title: string } = $props();
</script>

<section>
  <header class="top"><h1>{title}</h1></header>
  <p class="empty">This page is coming soon.</p>
</section>

<style lang="scss">
  .top { padding: 1rem; border-bottom: 1px solid var(--border); }
  h1 { margin: 0; font-size: 1.25rem; }
  .empty { padding: 1.5rem 1rem; color: var(--text-muted); }
</style>