<script lang="ts">
  import { onMount } from 'svelte';
  import { userApi, type FollowUserItem } from '../api';
  import { currentUser } from '../user';
  import { navigate } from '../router';

  let {
    targetUserId,
    targetUsername,
    initialTab = 'followers',
    onClose,
    onFollowChange,
  }: {
    targetUserId: string;
    targetUsername: string;
    initialTab?: 'followers' | 'following';
    onClose: () => void;
    onFollowChange?: (targetId: string, isNowFollowing: boolean) => void;
  } = $props();

  let activeTab = $state<'followers' | 'following'>(initialTab);
  let users = $state<FollowUserItem[]>([]);
  let nextCursor = $state('');
  let hasMore = $state(false);
  let loading = $state(true);
  let loadingMore = $state(false);
  let errorMessage = $state('');
  let actionLoading = $state<Record<string, boolean>>({});

  let currentRequestId = 0;
  let modalRef: HTMLDivElement;

  async function loadData(tab: 'followers' | 'following', cursor = '', isAppend = false) {
    const reqId = ++currentRequestId;
    if (!isAppend) {
      loading = true;
      errorMessage = '';
    } else {
      loadingMore = true;
    }

    try {
      let res;
      if (tab === 'followers') {
        res = await userApi.getFollowers(targetUserId, cursor);
      } else {
        res = await userApi.getFollowing(targetUserId, cursor);
      }

      if (reqId !== currentRequestId) return;

      if (isAppend) {
        users = [...users, ...res.users];
      } else {
        users = res.users;
      }
      nextCursor = res.nextCursor;
      hasMore = res.hasMore;

      if (res.users.length === 0 && res.hasMore && res.nextCursor) {
        loadData(tab, res.nextCursor, true);
      }
    } catch (err: any) {
      if (reqId === currentRequestId) {
        errorMessage = err.message || 'Failed to load list';
      }
    } finally {
      if (reqId === currentRequestId) {
        loading = false;
        loadingMore = false;
      }
    }
  }

  function handleTabChange(tab: 'followers' | 'following') {
    if (activeTab === tab) return;
    activeTab = tab;
    users = [];
    nextCursor = '';
    hasMore = false;
    loadData(tab);
  }

  function handleLoadMore() {
    if (loadingMore || !hasMore || !nextCursor) return;
    loadData(activeTab, nextCursor, true);
  }

  async function toggleFollow(item: FollowUserItem) {
    if (actionLoading[item.userId]) return;

    const previousStatus = item.isFollowing;
    item.isFollowing = !previousStatus;
    actionLoading[item.userId] = true;
    errorMessage = '';

    try {
      if (previousStatus) {
        await userApi.unfollow(item.userId);
      } else {
        await userApi.follow(item.userId);
      }
      onFollowChange?.(item.userId, item.isFollowing);
    } catch (err: any) {
      item.isFollowing = previousStatus;
      errorMessage = err.message || 'Failed to update follow status';
    } finally {
      actionLoading[item.userId] = false;
    }
  }

  function handleUserClick(uname: string) {
    onClose();
    navigate(`/profile/${uname}`);
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    }
  }

  onMount(() => {
    loadData(activeTab);
    modalRef?.focus();
  });
</script>

<svelte:window onkeydown={handleKeyDown} />

<div class="modal-backdrop" onclick={onClose} role="presentation">
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="modal-card"
    bind:this={modalRef}
    tabindex="-1"
    onclick={(e) => e.stopPropagation()}
    role="dialog"
    aria-modal="true"
    aria-label={`${targetUsername}'s ${activeTab}`}
  >
    <div class="modal-header">
      <div class="header-top">
        <button class="btn-close" type="button" onclick={onClose} aria-label="Close">
          ✕
        </button>
        <span class="header-title">@{targetUsername}</span>
      </div>

      <div class="tabs-nav" role="tablist">
        <button
          type="button"
          role="tab"
          class="tab-btn"
          class:active={activeTab === 'followers'}
          onclick={() => handleTabChange('followers')}
        >
          Followers
        </button>
        <button
          type="button"
          role="tab"
          class="tab-btn"
          class:active={activeTab === 'following'}
          onclick={() => handleTabChange('following')}
        >
          Following
        </button>
      </div>
    </div>

    {#if errorMessage}
      <div class="alert alert-error" role="alert">
        {errorMessage}
      </div>
    {/if}

    <div class="modal-body">
      {#if loading}
        <div class="skeleton-list">
          {#each Array(4) as _}
            <div class="skeleton-row">
              <div class="sk-avatar"></div>
              <div class="sk-lines">
                <div class="sk-line sk-title"></div>
                <div class="sk-line sk-sub"></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if users.length === 0}
        <div class="empty-state">
          {#if activeTab === 'followers'}
            <p class="empty-title">No followers yet</p>
            <p class="empty-desc">When people follow @{targetUsername}, they’ll show up here.</p>
          {:else}
            <p class="empty-title">Not following anyone yet</p>
            <p class="empty-desc">When @{targetUsername} follows accounts, they’ll show up here.</p>
          {/if}
        </div>
      {:else}
        <ul class="user-list">
          {#each users as u (u.userId)}
            <li class="user-item">
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="user-info" onclick={() => handleUserClick(u.username)}>
                {#if u.profilePictureUrl}
                  <img src={u.profilePictureUrl} alt={u.name} class="avatar" />
                {:else}
                  <div class="avatar-fallback">{u.name.charAt(0) || 'U'}</div>
                {/if}
                <div class="names">
                  <span class="display-name">{u.name}</span>
                  <span class="user-handle">@{u.username}</span>
                </div>
              </div>

              {#if u.userId !== $currentUser?.userId}
                <button
                  type="button"
                  class="btn-follow"
                  class:following={u.isFollowing}
                  disabled={actionLoading[u.userId]}
                  onclick={() => toggleFollow(u)}
                >
                  {u.isFollowing ? 'Following' : 'Follow'}
                </button>
              {/if}
            </li>
          {/each}
        </ul>

        {#if hasMore}
          <div class="load-more-wrap">
            <button
              type="button"
              class="btn-load-more"
              disabled={loadingMore}
              onclick={handleLoadMore}
            >
              {loadingMore ? 'Loading...' : 'Load more'}
            </button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>

<style lang="scss">
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 16px;
  }

  .modal-card {
    background: var(--bg-surface, #ffffff);
    color: var(--text-primary, #0f1419);
    width: 100%;
    max-width: 560px;
    height: 600px;
    max-height: 90vh;
    border-radius: 16px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.3);
    outline: none;

    @media (max-width: 640px) {
      max-width: 100%;
      height: 100vh;
      max-height: 100vh;
      border-radius: 0;
    }
  }

  .modal-header {
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--border-color, #eff3f4);
    background: var(--bg-surface, #ffffff);

    .header-top {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 12px 16px;

      .header-title {
        font-weight: 700;
        font-size: 1.1rem;
      }
    }
  }

  .btn-close {
    background: transparent;
    border: none;
    font-size: 1.1rem;
    cursor: pointer;
    color: inherit;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background: var(--hover-overlay, rgba(0, 0, 0, 0.05));
    }
  }

  .tabs-nav {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border-top: 1px solid var(--border-color, #eff3f4);

    .tab-btn {
      background: transparent;
      border: none;
      padding: 14px 0;
      font-size: 0.95rem;
      font-weight: 600;
      color: var(--text-secondary, #536471);
      cursor: pointer;
      position: relative;
      transition: color 0.2s;

      &:hover {
        background: var(--hover-overlay, rgba(0, 0, 0, 0.03));
        color: var(--text-primary, #0f1419);
      }

      &.active {
        color: var(--text-primary, #0f1419);
        font-weight: 700;

        &::after {
          content: '';
          position: absolute;
          bottom: 0;
          left: 50%;
          transform: translateX(-50%);
          width: 56px;
          height: 4px;
          border-radius: 9999px;
          background: var(--primary, #1d9bf0);
        }
      }
    }
  }

  .alert-error {
    background: rgba(244, 33, 46, 0.1);
    color: #f4212e;
    padding: 10px 16px;
    font-size: 0.9rem;
    border-left: 4px solid #f4212e;
  }

  .modal-body {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  .user-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .user-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    transition: background 0.2s;

    &:hover {
      background: var(--hover-overlay, rgba(0, 0, 0, 0.03));
    }

    .user-info {
      display: flex;
      align-items: center;
      gap: 12px;
      cursor: pointer;
      flex: 1;
      min-width: 0;

      .avatar {
        width: 44px;
        height: 44px;
        border-radius: 50%;
        object-fit: cover;
      }

      .avatar-fallback {
        width: 44px;
        height: 44px;
        border-radius: 50%;
        background: var(--bg-secondary, #cfd9de);
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 1.1rem;
        color: var(--text-secondary, #536471);
      }

      .names {
        display: flex;
        flex-direction: column;
        overflow: hidden;

        .display-name {
          font-weight: 700;
          font-size: 0.95rem;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;

          &:hover {
            text-decoration: underline;
          }
        }

        .user-handle {
          font-size: 0.85rem;
          color: var(--text-secondary, #536471);
        }
      }
    }
  }

  .btn-follow {
    background: var(--text-primary, #0f1419);
    color: var(--bg-surface, #ffffff);
    border: 1px solid transparent;
    border-radius: 9999px;
    padding: 6px 16px;
    font-weight: 700;
    font-size: 0.85rem;
    cursor: pointer;
    transition: background 0.2s;
    flex-shrink: 0;

    &:hover:not(:disabled) {
      opacity: 0.9;
    }

    &.following {
      background: transparent;
      color: var(--text-primary, #0f1419);
      border-color: var(--border-color, #cfd9de);

      &:hover:not(:disabled) {
        border-color: #f4212e;
        color: #f4212e;
        background: rgba(244, 33, 46, 0.08);
      }
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .load-more-wrap {
    padding: 16px;
    display: flex;
    justify-content: center;

    .btn-load-more {
      background: transparent;
      border: 1px solid var(--border-color, #cfd9de);
      color: var(--text-primary, #0f1419);
      padding: 8px 20px;
      border-radius: 9999px;
      font-weight: 600;
      font-size: 0.9rem;
      cursor: pointer;

      &:hover:not(:disabled) {
        background: var(--hover-overlay, rgba(0, 0, 0, 0.04));
      }
    }
  }

  .empty-state {
    padding: 48px 24px;
    text-align: center;

    .empty-title {
      font-weight: 700;
      font-size: 1.25rem;
      margin: 0 0 8px;
    }

    .empty-desc {
      color: var(--text-secondary, #536471);
      font-size: 0.95rem;
      margin: 0;
    }
  }

  .skeleton-list {
    padding: 8px 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .skeleton-row {
    display: flex;
    align-items: center;
    gap: 12px;

    .sk-avatar {
      width: 44px;
      height: 44px;
      border-radius: 50%;
      background: var(--border-color, #eff3f4);
    }

    .sk-lines {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 6px;

      .sk-line {
        height: 12px;
        border-radius: 4px;
        background: var(--border-color, #eff3f4);
      }

      .sk-title {
        width: 140px;
      }

      .sk-sub {
        width: 90px;
      }
    }
  }
</style>