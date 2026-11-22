<script lang="ts">
  import { onMount } from 'svelte';
  import { threadApi, type ThreadItem } from './api';
  import CreateThread from './components/CreateThread.svelte';
  import ThreadCard from './components/ThreadCard.svelte';
  import ThreadSkeleton from './components/ThreadSkeleton.svelte';
  import { navigate } from './router';

  let { threadId }: { threadId: string } = $props();

  let mainThread = $state<ThreadItem | null>(null);
  let parentThread = $state<ThreadItem | null>(null);
  let replies = $state<ThreadItem[]>([]);
  let loading = $state(true);
  let errorMsg = $state('');

  async function loadData() {
    if (!threadId) return;
    loading = true;
    errorMsg = '';
    try {
      const t = await threadApi.getById(threadId);
      mainThread = t;

      if (t.parentId) {
        try {
          parentThread = await threadApi.getById(t.parentId);
        } catch {
          parentThread = null;
        }
      } else {
        parentThread = null;
      }

      const repRes = await threadApi.replies(threadId);
      replies = repRes.replies ?? [];
    } catch (err: any) {
      errorMsg = err.message || 'Failed to load thread';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
  });

  $effect(() => {
    if (threadId) {
      loadData();
    }
  });

  function handleReplyCreated(newReply: ThreadItem) {
    replies = [...replies, newReply];
    if (mainThread) {
      mainThread.replyCount++;
    }
  }

  function handleThreadDeleted(id: string) {
    if (id === mainThread?.id) {
      navigate('/home');
    } else {
      replies = replies.filter((r) => r.id !== id);
      if (mainThread && mainThread.replyCount > 0) {
        mainThread.replyCount--;
      }
    }
  }
</script>

<div class="thread-detail-container">
  <header class="detail-header">
    <button type="button" class="btn-back" onclick={() => window.history.back()} aria-label="Go back">
      <svg viewBox="0 0 24 24" width="20" height="20">
        <path fill="currentColor" d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z" />
      </svg>
    </button>
    <h1>Thread</h1>
  </header>

  {#if loading}
    <div class="skeleton-wrap">
      <ThreadSkeleton />
      <ThreadSkeleton />
    </div>
  {:else if errorMsg}
    <div class="error-state">
      <p>{errorMsg}</p>
      <button class="btn-retry" onclick={loadData}>Retry</button>
    </div>
  {:else if mainThread}
    <div class="thread-flow">
      {#if parentThread}
        <div class="parent-thread-wrap">
          <ThreadCard thread={parentThread} onDelete={handleThreadDeleted} />
          <div class="connector-line"></div>
        </div>
      {/if}

      <div class="main-thread-card">
        <ThreadCard thread={mainThread} onDelete={handleThreadDeleted} />
      </div>

      <div class="reply-composer">
        <div class="replying-to">
          Replying to <span class="highlight">@{mainThread.authorUsername}</span>
        </div>
        <CreateThread
          parentId={mainThread.id}
          placeholder="Post your reply"
          onCreated={handleReplyCreated}
        />
      </div>

      <section class="replies-list">
        {#if replies.length === 0}
          <div class="empty-replies">
            <p>No replies yet. Be the first to reply!</p>
          </div>
        {:else}
          {#each replies as reply (reply.id)}
            <div class="reply-card-wrap">
              <ThreadCard thread={reply} onDelete={handleThreadDeleted} />
            </div>
          {/each}
        {/if}
      </section>
    </div>
  {/if}
</div>

<style lang="scss">
  .thread-detail-container {
    width: 100%;
    min-height: 100vh;
  }

  .detail-header {
    position: sticky;
    top: 0;
    z-index: 20;
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 0.75rem 1rem;
    background: var(--bg);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--border);

    h1 {
      font-size: 1.25rem;
      font-weight: 700;
      margin: 0;
    }
  }

  .btn-back {
    background: transparent;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    cursor: pointer;
    color: var(--text);
    transition: background-color 0.15s ease;

    &:hover {
      background-color: var(--border);
    }
  }

  .skeleton-wrap {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .error-state {
    padding: 3rem 1rem;
    text-align: center;
    color: var(--text-muted);

    .btn-retry {
      margin-top: 1rem;
      padding: 0.5rem 1.25rem;
      border-radius: 9999px;
      border: 1px solid var(--border);
      background: var(--surface);
      color: var(--text);
      cursor: pointer;
      font-weight: 600;

      &:hover {
        background: var(--border);
      }
    }
  }

  .thread-flow {
    display: flex;
    flex-direction: column;
  }

  .parent-thread-wrap {
    position: relative;
  }

  .connector-line {
    position: absolute;
    left: 35px;
    bottom: -1px;
    top: 55px;
    width: 2px;
    background-color: var(--border);
    z-index: 1;
  }

  .main-thread-card {
    border-bottom: 1px solid var(--border);
  }

  .reply-composer {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
    background: var(--bg);

    .replying-to {
      font-size: 0.85rem;
      color: var(--text-muted);
      margin-bottom: 0.5rem;
      margin-left: 3rem;

      .highlight {
        color: var(--accent);
      }
    }
  }

  .replies-list {
    display: flex;
    flex-direction: column;
  }

  .reply-card-wrap {
    border-bottom: 1px solid var(--border);
  }

  .empty-replies {
    padding: 3rem 1rem;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.95rem;
  }
</style>
