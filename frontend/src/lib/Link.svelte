<script lang="ts">
  import type { Snippet } from 'svelte';
  import { navigate } from './router';

  let { href, class: className = '', children }: {
    href: string;
    class?: string;
    children: Snippet;
  } = $props();

  function handleClick(e: MouseEvent) {
    if (e.ctrlKey || e.metaKey || e.shiftKey) return;
    e.preventDefault();
    navigate(href);
  }
</script>

<a {href} class={className} onclick={handleClick}>
  {@render children()}
</a>