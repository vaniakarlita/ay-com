<script lang="ts">
  import { onMount } from 'svelte';
  import { threadApi, type ThreadItem } from './api';
  import ThreadCard from './components/ThreadCard.svelte';
  import ThreadSkeleton from './components/ThreadSkeleton.svelte';

  let threads = $state<ThreadItem[]>([]);
  let searchQuery = $state('');
  let cursor = $state('');
  let hasMore = $state(true);
  let loading = $state(false);
  let initialLoaded = $state(false);
  let errorMsg = $state('');

  let sentinelElement = $state<HTMLDivElement | null>(null);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  async function loadBookmarks(reset = false) {
    if (loading) return;
    if (!reset && !hasMore) return;

    loading = true;
    errorMsg = '';
    const currentCursor = reset ? '' : cursor;

    try {
      const res = await threadApi.bookmarks(searchQuery, currentCursor, 15);
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
      errorMsg = err.message || 'Failed to load bookmarks';
      hasMore = false;
    } finally {
      loading = false;
      initialLoaded = true;
    }
  }

  function handleSearchInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    searchQuery = val;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      loadBookmarks(true);
    }, 300);
  }

  function handleBookmarkRemoved(removedId: string) {
    threads = threads.filter((t) => t.id !== removedId);
  }

  function handleThreadDeleted(deletedId: string) {
    threads = threads.filter((t) => t.id !== deletedId);
  }

  $effect(() => {
    const el = sentinelElement;
    if (!el) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          loadBookmarks();
        }
      },
      { rootMargin: '300px' }
    );
    observer.observe(el);

    return () => observer.disconnect();
  });

  onMount(() => {
    loadBookmarks(true);
  });
</script>

<section class="bookmarks-page">
  <header class="top">
    <div class="header-titles">
      <h1>Bookmarks</h1>
      <span class="sub-label">Saved threads</span>
    </div>

    <div class="search-wrap">
      <svg viewBox="0 0 24 24" class="search-icon">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        type="search"
        placeholder="Search bookmarks..."
        value={searchQuery}
        oninput={handleSearchInput}
      />
      {#if searchQuery}
        <button
          type="button"
          class="clear-btn"
          onclick={() => {
            searchQuery = '';
            loadBookmarks(true);
          }}
        >
          ✕
        </button>
      {/if}
    </div>
  </header>

  <div class="content-body">
    {#if errorMsg}
      <div class="error-box">
        <p>{errorMsg}</p>
        <button type="button" class="retry-btn" onclick={() => loadBookmarks(true)}>
          Retry
        </button>
      </div>
    {/if}

    {#if threads.length > 0}
      <div class="threads-list">
        {#each threads as thread (thread.id)}
          <ThreadCard
            {thread}
            onDelete={handleThreadDeleted}
            onBookmarkRemoved={handleBookmarkRemoved}
          />
        {/each}
      </div>
    {:else if initialLoaded && !loading && !errorMsg}
      <div class="empty-state">
        <p class="title">{searchQuery ? 'No bookmarks match your search' : 'Save threads for later'}</p>
        <p class="subtitle">
          {searchQuery
            ? 'Try searching for different keywords or clear the search input.'
            : 'Don’t let the good ones fly away! Bookmark threads to easily find them again in the future.'}
        </p>
      </div>
    {/if}

    {#if loading}
      <ThreadSkeleton count={initialLoaded ? 2 : 4} />
    {/if}

    <div bind:this={sentinelElement} class="sentinel"></div>
  </div>
</section>

<style lang="scss">
  .bookmarks-page {
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
    padding: 0.85rem 1rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .header-titles {
    display: flex;
    flex-direction: column;

    h1 {
      margin: 0;
      font-size: 1.25rem;
      font-weight: 800;
      line-height: 1.2;
    }

    .sub-label {
      font-size: 0.8rem;
      color: var(--text-muted);
    }
  }

  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;

    .search-icon {
      position: absolute;
      left: 0.85rem;
      width: 17px;
      height: 17px;
      fill: none;
      stroke: var(--text-muted);
      stroke-width: 2;
      stroke-linecap: round;
      stroke-linejoin: round;
      pointer-events: none;
    }

    input {
      width: 100%;
      padding: 0.55rem 2.2rem 0.55rem 2.5rem;
      border-radius: 999px;
      border: 1px solid var(--border);
      background: var(--border);
      color: var(--text);
      font-size: 0.9rem;
      outline: none;
      transition: border-color 0.15s ease, background 0.15s ease;

      &:focus {
        border-color: var(--primary);
        background: var(--bg);
      }
    }

    .clear-btn {
      position: absolute;
      right: 0.65rem;
      background: var(--text-muted);
      color: var(--bg);
      border: none;
      border-radius: 50%;
      width: 18px;
      height: 18px;
      display: grid;
      place-items: center;
      font-size: 0.65rem;
      cursor: pointer;
    }
  }

  .content-body {
    display: flex;
    flex-direction: column;
  }

  .threads-list {
    display: flex;
    flex-direction: column;
  }

  .empty-state {
    padding: 4rem 1.5rem;
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
      max-width: 340px;
      margin: 0 auto;
      line-height: 1.4;
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
