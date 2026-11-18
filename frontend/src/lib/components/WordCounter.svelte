<script lang="ts">
  let { count = 0, max = 280 }: { count: number; max?: number } = $props();

  const radius = 12;
  const circumference = 2 * Math.PI * radius;
  
  const percentage = $derived(Math.min(100, Math.max(0, (count / max) * 100)));
  const offset = $derived(circumference - (percentage / 100) * circumference);
  const isNearLimit = $derived(count >= max * 0.8 && count <= max);
  const isOverLimit = $derived(count > max);
</script>

<div class="counter-container" title="{count} / {max}">
  <svg class="ring" width="32" height="32" viewBox="0 0 32 32">
    <circle
      class="bg"
      cx="16"
      cy="16"
      r={radius}
    />
    <circle
      class="progress"
      class:near={isNearLimit}
      class:over={isOverLimit}
      cx="16"
      cy="16"
      r={radius}
      stroke-dasharray={circumference}
      stroke-dashoffset={offset}
    />
  </svg>
  {#if count >= max * 0.8}
    <span class="count-text" class:over={isOverLimit}>
      {max - count}
    </span>
  {/if}
</div>

<style lang="scss">
  .counter-container {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    position: relative;
  }

  .ring {
    transform: rotate(-90deg);
  }

  circle {
    fill: none;
    stroke-width: 2.5;
    transition: stroke-dashoffset 0.15s ease, stroke 0.15s ease;
  }

  .bg {
    stroke: var(--border);
  }

  .progress {
    stroke: var(--primary);
    stroke-linecap: round;

    &.near {
      stroke: #f59e0b; // Amber/Warning
    }

    &.over {
      stroke: #ef4444; // Red/Danger
    }
  }

  .count-text {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-muted);

    &.over {
      color: #ef4444;
    }
  }
</style>
