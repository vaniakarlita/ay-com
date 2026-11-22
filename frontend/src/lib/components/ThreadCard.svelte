<script lang="ts">
  import { threadApi, type ThreadItem } from '../api';
  import { navigate } from '../router';
  import { currentUser } from '../user';
  import { formatRelativeTime } from '../utils/time';
  import RichText from './RichText.svelte';

  let {
    thread,
    onDelete,
    onBookmarkRemoved,
  }: {
    thread: ThreadItem;
    onDelete?: (id: string) => void;
    onBookmarkRemoved?: (id: string) => void;
  } = $props();

  let showMenu = $state(false);
  let isDeleting = $state(false);
  let isInteracting = $state(false);

  let liked = $state(false);
  let likeCount = $state(0);
  let reposted = $state(false);
  let repostCount = $state(0);
  let bookmarked = $state(false);

  $effect(() => {
    liked = thread.likedByMe;
    likeCount = thread.likeCount;
    reposted = thread.repostedByMe;
    repostCount = thread.repostCount;
    bookmarked = thread.bookmarkedByMe;
  });

  const isOwner = $derived($currentUser?.userId === thread.userId);
  const initial = $derived((thread.authorName || 'U').charAt(0).toUpperCase());
  const relativeTime = $derived(formatRelativeTime(thread.createdAt));

  function goToDetail(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('a') || target.closest('.menu-dropdown')) {
      return;
    }
    navigate(`/thread/${thread.id}`);
  }

  function goToProfile(e: MouseEvent) {
    e.stopPropagation();
    navigate(`/profile?u=${encodeURIComponent(thread.authorUsername)}`);
  }

  async function toggleLike(e: MouseEvent) {
    e.stopPropagation();
    if (isInteracting) return;
    isInteracting = true;

    const previousLiked = liked;
    const previousCount = likeCount;

    liked = !liked;
    likeCount += liked ? 1 : -1;

    try {
      const res = previousLiked
        ? await threadApi.unlike(thread.id)
        : await threadApi.like(thread.id);
      liked = res.likedByMe;
      likeCount = res.likeCount;
    } catch {
      liked = previousLiked;
      likeCount = previousCount;
    } finally {
      isInteracting = false;
    }
  }

  async function toggleRepost(e: MouseEvent) {
    e.stopPropagation();
    if (isInteracting) return;
    isInteracting = true;

    const previousReposted = reposted;
    const previousCount = repostCount;

    reposted = !reposted;
    repostCount += reposted ? 1 : -1;

    try {
      const res = previousReposted
        ? await threadApi.unrepost(thread.id)
        : await threadApi.repost(thread.id);
      reposted = res.repostedByMe;
      repostCount = res.repostCount;
    } catch {
      reposted = previousReposted;
      repostCount = previousCount;
    } finally {
      isInteracting = false;
    }
  }

  async function toggleBookmark(e: MouseEvent) {
    e.stopPropagation();
    if (isInteracting) return;
    isInteracting = true;

    const previousBookmarked = bookmarked;
    bookmarked = !bookmarked;

    try {
      const res = previousBookmarked
        ? await threadApi.unbookmark(thread.id)
        : await threadApi.bookmark(thread.id);
      bookmarked = res.bookmarkedByMe;
      if (!bookmarked && onBookmarkRemoved) {
        onBookmarkRemoved(thread.id);
      }
    } catch {
      bookmarked = previousBookmarked;
    } finally {
      isInteracting = false;
    }
  }

  function toggleMenu(e: MouseEvent) {
    e.stopPropagation();
    showMenu = !showMenu;
  }

  async function handleDelete(e: MouseEvent) {
    e.stopPropagation();
    if (!confirm('Are you sure you want to delete this thread?')) return;

    isDeleting = true;
    try {
      await threadApi.delete(thread.id);
      showMenu = false;
      onDelete?.(thread.id);
    } catch {
      alert('Failed to delete thread. Please try again.');
    } finally {
      isDeleting = false;
    }
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<article
  class="thread-card"
  class:is-ad={thread.isAd}
  onclick={goToDetail}
>
  {#if thread.repostedByName}
    <div class="repost-header">
      <svg viewBox="0 0 24 24" class="svg-icon xs">
        <path d="M17 1l4 4-4 4" />
        <path d="M3 11V9a4 4 0 0 1 4-4h14" />
        <path d="M7 23l-4-4 4-4" />
        <path d="M21 13v2a4 4 0 0 1-4 4H3" />
      </svg>
      <span>{thread.repostedByName} reposted</span>
    </div>
  {/if}

  {#if thread.isAd}
    <div class="ad-badge-header">
      <span class="ad-label">Sponsored</span>
    </div>
  {/if}

  <div class="card-layout">
    <button type="button" class="avatar-btn" onclick={goToProfile} title={thread.authorName}>
      {#if thread.authorAvatarUrl}
        <img src={thread.authorAvatarUrl} alt="" class="avatar" />
      {:else}
        <div class="avatar fallback">{initial}</div>
      {/if}
    </button>

    <div class="content-col">
      <header class="header">
        <div class="author-info">
          <button type="button" class="name-btn" onclick={goToProfile}>
            <span class="name">{thread.authorName}</span>
            {#if thread.authorIsVerified}
              <span class="verified-badge" title="Verified User">✓</span>
            {/if}
          </button>
          <span class="handle">@{thread.authorUsername}</span>
          <span class="dot">·</span>
          <time class="time">{relativeTime}</time>
        </div>

        {#if isOwner}
          <div class="menu-container">
            <button
              type="button"
              class="icon-btn more-btn"
              aria-label="More options"
              onclick={toggleMenu}
            >
              <svg viewBox="0 0 24 24" class="svg-icon">
                <circle cx="5" cy="12" r="2" />
                <circle cx="12" cy="12" r="2" />
                <circle cx="19" cy="12" r="2" />
              </svg>
            </button>

            {#if showMenu}
              <div class="menu-dropdown">
                <button
                  type="button"
                  class="menu-item delete"
                  onclick={handleDelete}
                  disabled={isDeleting}
                >
                  <svg viewBox="0 0 24 24" class="svg-icon sm">
                    <path d="M3 6h18 M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6 M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  <span>{isDeleting ? 'Deleting...' : 'Delete thread'}</span>
                </button>
              </div>
            {/if}
          </div>
        {/if}
      </header>

      {#if thread.categories && thread.categories.length > 0}
        <div class="categories-row">
          {#each thread.categories as cat}
            <span class="cat-pill">{cat}</span>
          {/each}
        </div>
      {/if}

      <div class="body-text">
        <RichText text={thread.content} />
      </div>

      <footer class="actions">
        <button type="button" class="action-btn reply" aria-label="Reply">
          <span class="icon-wrap">
            <svg viewBox="0 0 24 24" class="svg-icon">
              <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
            </svg>
          </span>
          <span class="count">{thread.replyCount || ''}</span>
        </button>

        <button
          type="button"
          class="action-btn repost"
          class:active={reposted}
          aria-label="Repost"
          onclick={toggleRepost}
        >
          <span class="icon-wrap">
            <svg viewBox="0 0 24 24" class="svg-icon">
              <path d="M17 1l4 4-4 4" />
              <path d="M3 11V9a4 4 0 0 1 4-4h14" />
              <path d="M7 23l-4-4 4-4" />
              <path d="M21 13v2a4 4 0 0 1-4 4H3" />
            </svg>
          </span>
          <span class="count">{repostCount || ''}</span>
        </button>

        <button
          type="button"
          class="action-btn like"
          class:active={liked}
          aria-label="Like"
          onclick={toggleLike}
        >
          <span class="icon-wrap">
            <svg viewBox="0 0 24 24" class="svg-icon" class:filled={liked}>
              <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
            </svg>
          </span>
          <span class="count">{likeCount || ''}</span>
        </button>

        <button
          type="button"
          class="action-btn bookmark"
          class:active={bookmarked}
          aria-label="Bookmark"
          onclick={toggleBookmark}
        >
          <span class="icon-wrap">
            <svg viewBox="0 0 24 24" class="svg-icon" class:filled={bookmarked}>
              <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z" />
            </svg>
          </span>
        </button>
      </footer>
    </div>
  </div>
</article>

<style lang="scss">
  .thread-card {
    display: block;
    width: 100%;
    padding: 0.85rem 1rem;
    border-bottom: 1px solid var(--border);
    cursor: pointer;
    text-align: left;
    background: transparent;
    transition: background 0.15s ease;

    &:hover {
      background: rgba(0, 0, 0, 0.02);

      :global([data-theme='dark']) & {
        background: rgba(255, 255, 255, 0.02);
      }
    }

    &.is-ad {
      background: rgba(29, 155, 240, 0.03);
      border-left: 3px solid var(--primary);
    }
  }

  .repost-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.35rem;
    padding-left: 3.25rem;
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--text-muted);
  }

  .ad-badge-header {
    display: flex;
    margin-bottom: 0.35rem;
    padding-left: 3.25rem;
  }

  .ad-label {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--primary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .card-layout {
    display: flex;
    gap: 0.75rem;
  }

  .avatar-btn {
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    align-self: flex-start;
  }

  .avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;

    &.fallback {
      display: grid;
      place-items: center;
      background: var(--primary);
      color: #fff;
      font-weight: 800;
      font-size: 1rem;
    }
  }

  .content-col {
    flex: 1;
    min-width: 0;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    line-height: 1.25;
    margin-bottom: 0.25rem;
  }

  .author-info {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    flex-wrap: wrap;
    min-width: 0;
  }

  .name-btn {
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    font-weight: 700;
    color: var(--text);

    &:hover .name {
      text-decoration: underline;
    }
  }

  .name {
    font-size: 0.95rem;
  }

  .verified-badge {
    color: #1d9bf0;
    font-size: 0.85rem;
    font-weight: 900;
  }

  .handle,
  .dot,
  .time {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .categories-row {
    display: flex;
    gap: 0.35rem;
    flex-wrap: wrap;
    margin-top: 0.2rem;
    margin-bottom: 0.4rem;
  }

  .cat-pill {
    font-size: 0.75rem;
    padding: 0.15rem 0.5rem;
    border-radius: 999px;
    background: var(--border);
    color: var(--text-muted);
    font-weight: 600;
  }

  .body-text {
    font-size: 0.95rem;
    color: var(--text);
    margin: 0.35rem 0 0.65rem;
    word-break: break-word;
  }

  .actions {
    display: flex;
    justify-content: space-between;
    max-width: 420px;
    margin-top: 0.35rem;
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    background: none;
    border: none;
    padding: 0.35rem;
    border-radius: 999px;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 0.82rem;
    transition: color 0.15s ease, background 0.15s ease;

    .icon-wrap {
      display: grid;
      place-items: center;
      padding: 0.25rem;
      border-radius: 50%;
    }

    &:hover {
      &.reply {
        color: #1d9bf0;
        .icon-wrap { background: rgba(29, 155, 240, 0.1); }
      }
      &.repost {
        color: #10b981;
        .icon-wrap { background: rgba(16, 185, 129, 0.1); }
      }
      &.like {
        color: #f43f5e;
        .icon-wrap { background: rgba(244, 63, 94, 0.1); }
      }
      &.bookmark {
        color: #1d9bf0;
        .icon-wrap { background: rgba(29, 155, 240, 0.1); }
      }
    }

    &.repost.active {
      color: #10b981;
    }

    &.like.active {
      color: #f43f5e;
    }

    &.bookmark.active {
      color: #1d9bf0;
    }
  }

  .svg-icon {
    width: 18px;
    height: 18px;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;

    &.filled {
      fill: currentColor;
    }

    &.sm {
      width: 16px;
      height: 16px;
    }

    &.xs {
      width: 13px;
      height: 13px;
    }
  }

  .menu-container {
    position: relative;
  }

  .more-btn {
    background: none;
    border: none;
    padding: 0.25rem;
    border-radius: 50%;
    cursor: pointer;
    color: var(--text-muted);

    &:hover {
      background: var(--border);
      color: var(--text);
    }
  }

  .menu-dropdown {
    position: absolute;
    right: 0;
    top: 100%;
    min-width: 160px;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 12px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    z-index: 10;
    overflow: hidden;

    .menu-item {
      width: 100%;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.65rem 0.85rem;
      border: none;
      background: none;
      font-size: 0.85rem;
      font-weight: 600;
      cursor: pointer;
      text-align: left;
      color: var(--text);

      &:hover {
        background: var(--border);
      }

      &.delete {
        color: #ef4444;
      }
    }
  }
</style>
