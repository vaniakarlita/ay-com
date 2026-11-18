<script lang="ts">
  let { count = 3 }: { count?: number } = $props();
</script>

<div class="skeleton-list">
  {#each Array(count) as _}
    <div class="skeleton-card">
      <div class="avatar-skeleton"></div>
      <div class="content-skeleton">
        <div class="header-skeleton">
          <div class="line name"></div>
          <div class="line handle"></div>
          <div class="line time"></div>
        </div>
        <div class="body-skeleton">
          <div class="line full"></div>
          <div class="line full"></div>
          <div class="line half"></div>
        </div>
        <div class="actions-skeleton">
          <div class="circle"></div>
          <div class="circle"></div>
          <div class="circle"></div>
          <div class="circle"></div>
        </div>
      </div>
    </div>
  {/each}
</div>

<style lang="scss">
  .skeleton-list {
    display: flex;
    flex-direction: column;
  }

  .skeleton-card {
    display: flex;
    gap: 0.75rem;
    padding: 1rem;
    border-bottom: 1px solid var(--border);
  }

  .avatar-skeleton {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--border);
    flex-shrink: 0;
    animation: pulse 1.5s infinite ease-in-out;
  }

  .content-skeleton {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .header-skeleton {
    display: flex;
    gap: 0.5rem;
    align-items: center;

    .name { width: 90px; height: 14px; }
    .handle { width: 70px; height: 14px; }
    .time { width: 40px; height: 14px; }
  }

  .body-skeleton {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;

    .full { width: 100%; height: 12px; }
    .half { width: 60%; height: 12px; }
  }

  .actions-skeleton {
    display: flex;
    gap: 3rem;
    margin-top: 0.25rem;

    .circle {
      width: 18px;
      height: 18px;
      border-radius: 50%;
    }
  }

  .line,
  .circle {
    background: var(--border);
    border-radius: 4px;
    animation: pulse 1.5s infinite ease-in-out;
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 0.5;
    }
    50% {
      opacity: 1;
    }
  }
</style>
