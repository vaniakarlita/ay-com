<script lang="ts">
  import { theme, toggleTheme } from './theme';
</script>

<button
  class="toggle"
  onclick={toggleTheme}
  aria-label="Switch to {$theme === 'light' ? 'dark' : 'light'} theme"
>
  {$theme === 'light' ? 'Dark' : 'Light'}
</button>

<style lang="scss">
  .toggle {
    position: fixed;
    top: 1rem;
    right: 1rem;
    padding: 0.5rem 1rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--bg);
    color: var(--text);
    font-weight: 700;
    cursor: pointer;

    &:hover {
      background: var(--border);
    }
  }
</style>