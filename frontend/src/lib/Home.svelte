<script lang="ts">
  import { untrack } from 'svelte';
  import { threadApi, type ThreadItem } from './api';
  import CreateThread from './components/CreateThread.svelte';
  import ThreadCard from './components/ThreadCard.svelte';
  import ThreadSkeleton from './components/ThreadSkeleton.svelte';
  import { latestCreatedThread } from './stores/ui';

  let tab = $state<'for-you' | 'following'>('for-you');
  let threads = $state<ThreadItem[]>([]);
  let cursor = $state<string>('');
  let hasMore = $state<boolean>(true);
  let loading = $state<boolean>(false);
  let initialLoaded = $state<boolean>(false);
  let errorMsg = $state<string>('');

  let sentinelElement = $state<HTMLDivElement | null>(null);

  async function loadFeed(reset = false) {
    if (loading) return;
    if (!reset && !hasMore) return;

    loading = true;
    errorMsg = '';
    const currentCursor = reset ? '' : cursor;

    try {
      const res = await threadApi.feed(currentCursor, 15);
      if (reset) {
        threads = res.threads ?? [];
      } else {
        const existingIds = new Set(threads.map((t) => t.id));
        const newThreads = (res.threads ?? []).filter((t) => !existingIds.has(t.id));
        threads = [...threads, ...newThreads];
      }
      cursor = res.nextCursor;
      hasMore = res.hasMore;
    } catch (err: any) {
      errorMsg = err.message || 'Failed to load feed';
      hasMore = false;
    } finally {
      loading = false;
      initialLoaded = true;
    }
  }

  function handleThreadCreated(newThread: ThreadItem) {
    threads = [newThread, ...threads];
  }

  function handleThreadDeleted(deletedId: string) {
    threads = threads.filter((t) => t.id !== deletedId);
  }

  $effect(() => {
    const current = tab;
    untrack(() => {
      if (current === 'for-you') {
        loadFeed(true);
      } else {
        threads = [];
        hasMore = false;
      }
    });
  });

  $effect(() => {
    const created = $latestCreatedThread;
    if (created) {
      untrack(() => handleThreadCreated(created));
      latestCreatedThread.set(null);
    }
  });

  $effect(() => {
    const el = sentinelElement;
    if (!el) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          loadFeed();
        }
      },
      { rootMargin: '300px' }
    );
    observer.observe(el);

    return () => observer.disconnect();
  });
</script>

<section class="home-container">
  <header class="top">
    <h1>Home</h1>
    <div class="tabs" role="tablist">
      <button
        type="button"
        role="tab"
        class:active={tab === 'for-you'}
        aria-selected={tab === 'for-you'}
        onclick={() => (tab = 'for-you')}
      >
        For you
      </button>
      <button
        type="button"
        role="tab"
        class:active={tab === 'following'}
        aria-selected={tab === 'following'}
        onclick={() => (tab = 'following')}
      >
        Following
      </button>
    </div>
  </header>

  <div class="composer-section">
    <CreateThread onCreated={handleThreadCreated} placeholder="What's happening?" />
  </div>

  <div class="feed-content">
    {#if errorMsg}
      <div class="error-box">
        <p>{errorMsg}</p>
        <button type="button" class="retry-btn" onclick={() => loadFeed(true)}>
          Retry
        </button>
      </div>
    {/if}

    {#if tab === 'following'}
      <div class="empty-state">
        <p class="title">No posts yet</p>
        <p class="subtitle">When accounts you follow post threads, they'll show up here.</p>
      </div>
    {:else}
      {#if threads.length > 0}
        <div class="threads-list">
          {#each threads as thread (thread.id)}
            <ThreadCard {thread} onDelete={handleThreadDeleted} />
          {/each}
        </div>
      {:else if initialLoaded && !loading && !errorMsg}
        <div class="empty-state">
          <p class="title">Welcome to AY.com!</p>
          <p class="subtitle">There are no threads yet. Be the first to share something above!</p>
        </div>
      {/if}

      {#if loading}
        <ThreadSkeleton count={initialLoaded ? 2 : 5} />
      {/if}

      <div bind:this={sentinelElement} class="sentinel"></div>
    {/if}
  </div>
</section>

<style lang="scss">
  .home-container {
    display: flex;
    flex-direction: column;
    min-height: 100%;
  }

  .top {
    position: sticky;
    top: 0;
    background: var(--bg);
    border-bottom: 1px solid var(--border);
    backdrop-filter: blur(12px);
    z-index: 10;
  }

  h1 {
    margin: 0;
    padding: 0.85rem 1rem 0.4rem;
    font-size: 1.25rem;
    font-weight: 800;
  }

  .tabs {
    display: flex;
  }

  .tabs button {
    flex: 1;
    padding: 0.9rem;
    border: 0;
    border-bottom: 3px solid transparent;
    background: none;
    color: var(--text-muted);
    font-weight: 700;
    font-size: 0.95rem;
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;

    &:hover {
      background: var(--border);
    }

    &.active {
      color: var(--text);
      border-bottom-color: var(--primary);
    }
  }

  .composer-section {
    border-bottom: 8px solid var(--border);
  }

  .feed-content {
    display: flex;
    flex-direction: column;
  }

  .threads-list {
    display: flex;
    flex-direction: column;
  }

  .empty-state {
    padding: 3rem 1.5rem;
    text-align: center;
    color: var(--text-muted);

    .title {
      font-size: 1.25rem;
      font-weight: 800;
      color: var(--text);
      margin-bottom: 0.5rem;
    }

    .subtitle {
      font-size: 0.95rem;
      max-width: 320px;
      margin: 0 auto;
    }
  }

  .error-box {
    margin: 1.5rem;
    padding: 1rem;
    background: rgba(239, 68, 68, 0.08);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 12px;
    text-align: center;
    color: #ef4444;

    p {
      margin-bottom: 0.5rem;
      font-size: 0.9rem;
    }

    .retry-btn {
      padding: 0.35rem 1rem;
      background: #ef4444;
      color: #fff;
      border: none;
      border-radius: 999px;
      font-weight: 700;
      cursor: pointer;
    }
  }

  .sentinel {
    height: 30px;
    width: 100%;
  }
</style>