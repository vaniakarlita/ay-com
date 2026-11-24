<script lang="ts">
  import { onMount } from 'svelte';
  import { userApi, threadApi, type UserProfileResponse, type ThreadItem } from './api';
  import { currentUser } from './user';
  import ThreadCard from './components/ThreadCard.svelte';
  import ThreadSkeleton from './components/ThreadSkeleton.svelte';
  import EditProfileModal from './components/EditProfileModal.svelte';
  import FollowListModal from './components/FollowListModal.svelte';
  import ReportModal from './components/ReportModal.svelte';

  let { username }: { username?: string } = $props();

  let profile = $state<UserProfileResponse | null>(null);
  let threads = $state<ThreadItem[]>([]);
  let activeTab = $state<'posts' | 'replies' | 'likes' | 'media'>('posts');
  let loading = $state(true);
  let threadsLoading = $state(false);
  let errorMsg = $state('');
  let isFollowing = $state(false);
  let followInProgress = $state(false);
  let showEditModal = $state(false);
  let showFollowModal = $state(false);
  let followModalTab = $state<'followers' | 'following'>('followers');
  let showReportModal = $state(false);
  let showMenu = $state(false);
  let showBlockConfirm = $state(false);
  let blockInProgress = $state(false);

  const isOwnProfile = $derived(!username || username === $currentUser?.username);
  const targetUsername = $derived(username || $currentUser?.username || '');

  async function loadProfile() {
    if (!targetUsername) return;
    loading = true;
    errorMsg = '';
    showMenu = false;
    showBlockConfirm = false;
    try {
      const data = await userApi.getProfile(targetUsername);
      profile = data;
      isFollowing = data.isFollowing;
      if (!data.isBlockedBy) {
        loadTabContent();
      }
    } catch (err: any) {
      errorMsg = err.message || 'User not found';
    } finally {
      loading = false;
    }
  }

  async function loadTabContent() {
    if (!profile) return;
    threadsLoading = true;
    try {
      let res;
      if (activeTab === 'posts') {
        res = await threadApi.userThreads(profile.userId);
      } else if (activeTab === 'replies') {
        res = await threadApi.userReplies(profile.userId);
      } else if (activeTab === 'likes') {
        res = await threadApi.userLikes(profile.userId);
      } else if (activeTab === 'media') {
        res = await threadApi.userMedia(profile.userId);
      }
      threads = res?.threads ?? [];
    } catch {
      threads = [];
    } finally {
      threadsLoading = false;
    }
  }

  function handleTabChange(tab: 'posts' | 'replies' | 'likes' | 'media') {
    if (activeTab === tab) return;
    activeTab = tab;
    loadTabContent();
  }

  function handlePinToggled(threadId: string, isPinned: boolean) {
    if (activeTab === 'posts' || activeTab === 'replies') {
      loadTabContent();
    }
  }

  async function toggleFollow() {
    if (!profile || followInProgress || profile.isBlocking || profile.isBlockedBy) return;
    followInProgress = true;
    try {
      if (isFollowing) {
        await userApi.unfollow(profile.userId);
        isFollowing = false;
        profile.followerCount = Math.max(0, profile.followerCount - 1);
      } else {
        await userApi.follow(profile.userId);
        isFollowing = true;
        profile.followerCount++;
      }
    } catch {
    } finally {
      followInProgress = false;
    }
  }

  async function handleBlock() {
    if (!profile || blockInProgress) return;
    blockInProgress = true;
    try {
      await userApi.block(profile.userId);
      profile.isBlocking = true;
      isFollowing = false;
      showBlockConfirm = false;
      showMenu = false;
      loadProfile();
    } catch {
    } finally {
      blockInProgress = false;
    }
  }

  async function handleUnblock() {
    if (!profile || blockInProgress) return;
    blockInProgress = true;
    try {
      await userApi.unblock(profile.userId);
      profile.isBlocking = false;
      showMenu = false;
      loadProfile();
    } catch {
    } finally {
      blockInProgress = false;
    }
  }

  function handleThreadDeleted(deletedId: string) {
    threads = threads.filter((t) => t.id !== deletedId);
  }

  onMount(() => {
    loadProfile();
  });

  $effect(() => {
    if (targetUsername) {
      loadProfile();
    }
  });

  function formatJoinDate(dateStr?: string): string {
    if (!dateStr) return '';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
    } catch {
      return '';
    }
  }
  function handleProfileUpdated(updatedUser: any) {
    if (profile) {
      profile.name = updatedUser.name;
      profile.username = updatedUser.username;
      profile.profilePictureUrl = updatedUser.profilePictureUrl;
      profile.bannerUrl = updatedUser.bannerUrl;
      profile.bio = updatedUser.bio;
    }
  }

  function openFollowModal(tab: 'followers' | 'following') {
    followModalTab = tab;
    showFollowModal = true;
  }

  function handleFollowChangeInModal(targetId: string, isNowFollowing: boolean) {
    if (!profile) return;
    if (isOwnProfile) {
      profile.followingCount = Math.max(0, profile.followingCount + (isNowFollowing ? 1 : -1));
    } else if (targetId === profile.userId) {
      isFollowing = isNowFollowing;
      profile.followerCount = Math.max(0, profile.followerCount + (isNowFollowing ? 1 : -1));
    }
  }
</script>

<div class="profile-container">
  {#if loading}
    <div class="skeleton-wrap">
      <ThreadSkeleton />
    </div>
  {:else if errorMsg}
    <div class="error-wrap">
      <h2>{errorMsg}</h2>
      <p>This account doesn’t exist or has been deactivated.</p>
    </div>
  {:else if profile}
    <header class="profile-header">
      <div class="banner">
        {#if profile.bannerUrl}
          <img src={profile.bannerUrl} alt="Profile banner" class="banner-img" />
        {:else}
          <div class="banner-placeholder"></div>
        {/if}
      </div>

      <div class="header-main">
        <div class="avatar-and-actions">
          <div class="avatar-wrap">
            {#if profile.profilePictureUrl}
              <img src={profile.profilePictureUrl} alt={profile.name} class="avatar-img" />
            {:else}
              <div class="avatar-fallback">{profile.name.charAt(0).toUpperCase()}</div>
            {/if}
          </div>

          <div class="actions">
            {#if isOwnProfile}
              <button class="btn btn-outline" type="button" onclick={() => (showEditModal = true)}>
                Edit profile
              </button>
            {:else if profile.isBlockedBy}
              <!-- Viewer is blocked: no follow button, only overflow menu if needed -->
              <div class="menu-container">
                <button
                  class="btn-icon"
                  type="button"
                  aria-label="More options"
                  onclick={() => (showMenu = !showMenu)}
                >
                  <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                    <circle cx="5" cy="12" r="2" />
                    <circle cx="12" cy="12" r="2" />
                    <circle cx="19" cy="12" r="2" />
                  </svg>
                </button>
                {#if showMenu}
                  <div class="dropdown-menu">
                    {#if profile.isBlocking}
                      <button type="button" class="menu-item" onclick={handleUnblock}>
                        Unblock @{profile.username}
                      </button>
                    {:else}
                      <button type="button" class="menu-item danger" onclick={() => { showMenu = false; showBlockConfirm = true; }}>
                        Block @{profile.username}
                      </button>
                    {/if}
                    <button type="button" class="menu-item danger" onclick={() => { showMenu = false; showReportModal = true; }}>
                      Report @{profile.username}
                    </button>
                  </div>
                {/if}
              </div>
            {:else if profile.isBlocking}
              <button
                class="btn btn-blocked"
                disabled={blockInProgress}
                onclick={handleUnblock}
                type="button"
              >
                <span class="label-blocked">Blocked</span>
                <span class="label-unblock">Unblock</span>
              </button>
              <div class="menu-container">
                <button
                  class="btn-icon"
                  type="button"
                  aria-label="More options"
                  onclick={() => (showMenu = !showMenu)}
                >
                  <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                    <circle cx="5" cy="12" r="2" />
                    <circle cx="12" cy="12" r="2" />
                    <circle cx="19" cy="12" r="2" />
                  </svg>
                </button>
                {#if showMenu}
                  <div class="dropdown-menu">
                    <button type="button" class="menu-item danger" onclick={() => { showMenu = false; showReportModal = true; }}>
                      Report @{profile.username}
                    </button>
                  </div>
                {/if}
              </div>
            {:else}
              <button
                class="btn btn-follow"
                class:following={isFollowing}
                disabled={followInProgress}
                onclick={toggleFollow}
                type="button"
              >
                {isFollowing ? 'Following' : 'Follow'}
              </button>
              <div class="menu-container">
                <button
                  class="btn-icon"
                  type="button"
                  aria-label="More options"
                  onclick={() => (showMenu = !showMenu)}
                >
                  <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                    <circle cx="5" cy="12" r="2" />
                    <circle cx="12" cy="12" r="2" />
                    <circle cx="19" cy="12" r="2" />
                  </svg>
                </button>
                {#if showMenu}
                  <div class="dropdown-menu">
                    <button type="button" class="menu-item danger" onclick={() => { showMenu = false; showBlockConfirm = true; }}>
                      Block @{profile.username}
                    </button>
                    <button type="button" class="menu-item danger" onclick={() => { showMenu = false; showReportModal = true; }}>
                      Report @{profile.username}
                    </button>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>

        <div class="user-meta">
          <h1 class="name">{profile.name}</h1>
          <span class="username">@{profile.username}</span>

          {#if profile.bio}
            <p class="bio">{profile.bio}</p>
          {/if}

          {#if profile.createdAt}
            <div class="join-date">
              <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
                <path fill="currentColor" d="M7 4V2h2v2h6V2h2v2h4v18H3V4h4zm12 6H5v10h14V10zm-2 2v2h-2v-2h2zm-4 0v2h-2v-2h2zm-4 0v2H7v-2h2z" />
              </svg>
              <span>Joined {formatJoinDate(profile.createdAt)}</span>
            </div>
          {/if}

          {#if !profile.isBlockedBy}
            <div class="stats">
              <button class="stat-btn" type="button" onclick={() => openFollowModal('following')}>
                <strong>{profile.followingCount}</strong> <span class="label">Following</span>
              </button>
              <button class="stat-btn" type="button" onclick={() => openFollowModal('followers')}>
                <strong>{profile.followerCount}</strong> <span class="label">Followers</span>
              </button>
            </div>
          {/if}
        </div>
      </div>

      {#if !profile.isBlockedBy}
        <div class="tabs" role="tablist">
          <button
            type="button"
            role="tab"
            class:active={activeTab === 'posts'}
            onclick={() => handleTabChange('posts')}
          >
            Posts
          </button>
          <button
            type="button"
            role="tab"
            class:active={activeTab === 'replies'}
            onclick={() => handleTabChange('replies')}
          >
            Replies
          </button>
          {#if isOwnProfile}
            <button
              type="button"
              role="tab"
              class:active={activeTab === 'likes'}
              onclick={() => handleTabChange('likes')}
            >
              Likes
            </button>
          {/if}
          <button
            type="button"
            role="tab"
            class:active={activeTab === 'media'}
            onclick={() => handleTabChange('media')}
          >
            Media
          </button>
        </div>
      {/if}
    </header>

    <main class="profile-content">
      {#if profile.isBlockedBy}
        <div class="blocked-notice">
          <h2>@{profile.username} blocked you</h2>
          <p>You are blocked from following @{profile.username} and viewing @{profile.username}'s posts.</p>
        </div>
      {:else if threadsLoading}
        <div class="skeleton-wrap">
          <ThreadSkeleton />
          <ThreadSkeleton />
        </div>
      {:else if threads.length === 0}
        <div class="empty-state">
          {#if activeTab === 'posts'}
            <p>No posts yet.</p>
          {:else if activeTab === 'replies'}
            <p>No replies yet.</p>
          {:else if activeTab === 'likes'}
            <p>No liked threads yet.</p>
          {:else}
            <p>No media yet.</p>
          {/if}
        </div>
      {:else}
        {#each threads as thread (thread.id)}
          <ThreadCard
            {thread}
            canPin={isOwnProfile && (activeTab === 'posts' || activeTab === 'replies')}
            onDelete={handleThreadDeleted}
            onPinToggled={handlePinToggled}
          />
        {/each}
      {/if}
    </main>
  {/if}

  {#if showEditModal && profile}
    <EditProfileModal
      user={{
        userId: profile.userId,
        name: profile.name,
        username: profile.username,
        profilePictureUrl: profile.profilePictureUrl,
        bannerUrl: profile.bannerUrl,
        bio: profile.bio,
      }}
      onClose={() => (showEditModal = false)}
      onSaved={handleProfileUpdated}
    />
  {/if}

  {#if showFollowModal && profile}
    <FollowListModal
      targetUserId={profile.userId}
      targetUsername={profile.username}
      initialTab={followModalTab}
      onClose={() => (showFollowModal = false)}
      onFollowChange={handleFollowChangeInModal}
    />
  {/if}

  {#if showReportModal && profile}
    <ReportModal
      targetUserId={profile.userId}
      targetUsername={profile.username}
      onClose={() => (showReportModal = false)}
    />
  {/if}

  {#if showBlockConfirm && profile}
    <div
      class="modal-backdrop"
      onclick={(e) => {
        if (e.target === e.currentTarget && !blockInProgress) showBlockConfirm = false;
      }}
      role="presentation"
    >
      <div class="confirm-card" role="dialog" aria-modal="true">
        <h3>Block @{profile.username}?</h3>
        <p>
          They will not be able to follow you or view your posts, and you will not see their posts or notifications. Mutual follows will also be removed.
        </p>
        <div class="confirm-actions">
          <button
            type="button"
            class="btn-cancel"
            disabled={blockInProgress}
            onclick={() => (showBlockConfirm = false)}
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn-confirm-block"
            disabled={blockInProgress}
            onclick={handleBlock}
          >
            {blockInProgress ? 'Blocking...' : 'Block'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style lang="scss">
  .profile-container {
    width: 100%;
    min-height: 100vh;
  }

  .banner {
    width: 100%;
    height: 160px;
    background: var(--border);
    position: relative;
    overflow: hidden;

    .banner-img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .banner-placeholder {
      width: 100%;
      height: 100%;
      background: linear-gradient(135deg, rgba(29, 155, 240, 0.2), rgba(29, 155, 240, 0.05));
    }
  }

  .header-main {
    padding: 0 1rem 1rem 1rem;
    position: relative;
  }

  .avatar-and-actions {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-top: -45px;
    margin-bottom: 0.75rem;
  }

  .avatar-wrap {
    width: 90px;
    height: 90px;
    border-radius: 50%;
    border: 4px solid var(--bg);
    overflow: hidden;
    background: var(--surface);
    flex-shrink: 0;

    .avatar-img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .avatar-fallback {
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 2.25rem;
      font-weight: 700;
      color: var(--text-muted);
    }
  }

  .actions {
    margin-bottom: 0.25rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;

    .btn {
      border-radius: 9999px;
      padding: 0.5rem 1.25rem;
      font-weight: 700;
      font-size: 0.95rem;
      cursor: pointer;
      transition: all 0.15s ease;
    }

    .btn-outline {
      border: 1px solid var(--border);
      background: transparent;
      color: var(--text);

      &:hover {
        background: rgba(255, 255, 255, 0.05);
      }
    }

    .btn-follow {
      border: 1px solid var(--text);
      background: var(--text);
      color: var(--bg);

      &.following {
        background: transparent;
        color: var(--text);
        border-color: var(--border);

        &:hover {
          border-color: #ef4444;
          color: #ef4444;
          background: rgba(239, 68, 68, 0.1);
        }
      }
    }

    .btn-blocked {
      border: 1px solid #ef4444;
      background: transparent;
      color: #ef4444;

      .label-unblock {
        display: none;
      }

      &:hover {
        background: rgba(239, 68, 68, 0.15);

        .label-blocked {
          display: none;
        }

        .label-unblock {
          display: inline;
        }
      }
    }

    .menu-container {
      position: relative;
    }

    .btn-icon {
      background: transparent;
      border: 1px solid var(--border);
      color: var(--text);
      border-radius: 50%;
      width: 36px;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      transition: background 0.15s;

      &:hover {
        background: rgba(255, 255, 255, 0.08);
      }
    }

    .dropdown-menu {
      position: absolute;
      right: 0;
      top: 110%;
      background: var(--bg-card, #1e2029);
      border: 1px solid var(--border);
      border-radius: 10px;
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
      min-width: 170px;
      z-index: 100;
      overflow: hidden;
      display: flex;
      flex-direction: column;

      .menu-item {
        background: transparent;
        border: none;
        padding: 10px 14px;
        text-align: left;
        color: var(--text);
        font-size: 0.9rem;
        cursor: pointer;
        transition: background 0.15s;

        &:hover {
          background: rgba(255, 255, 255, 0.06);
        }

        &.danger {
          color: #ef4444;

          &:hover {
            background: rgba(239, 68, 68, 0.1);
          }
        }
      }
    }
  }

  .user-meta {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;

    .name {
      margin: 0;
      font-size: 1.35rem;
      font-weight: 800;
    }

    .username {
      color: var(--text-muted);
      font-size: 0.95rem;
    }

    .bio {
      margin: 0.5rem 0 0.25rem 0;
      font-size: 0.95rem;
      line-height: 1.4;
      white-space: pre-wrap;
    }

    .join-date {
      display: flex;
      align-items: center;
      gap: 0.4rem;
      color: var(--text-muted);
      font-size: 0.85rem;
      margin-top: 0.25rem;
    }

    .stats {
      display: flex;
      gap: 1.25rem;
      margin-top: 0.5rem;
      font-size: 0.9rem;

      .stat-btn {
        background: transparent;
        border: none;
        padding: 0;
        cursor: pointer;
        font-size: inherit;
        color: inherit;
        display: inline-flex;
        gap: 4px;

        strong {
          color: var(--text);
        }

        .label {
          color: var(--text-muted);
        }

        &:hover strong,
        &:hover .label {
          text-decoration: underline;
        }
      }
    }
  }

  .tabs {
    display: flex;
    border-bottom: 1px solid var(--border);
    margin-top: 1rem;

    button {
      flex: 1;
      padding: 0.85rem 0;
      background: transparent;
      border: none;
      color: var(--text-muted);
      font-weight: 600;
      font-size: 0.95rem;
      cursor: pointer;
      position: relative;

      &:hover {
        background: rgba(255, 255, 255, 0.03);
        color: var(--text);
      }

      &.active {
        color: var(--text);
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
          background: var(--accent);
        }
      }
    }
  }

  .empty-state {
    padding: 3rem 1rem;
    text-align: center;
    color: var(--text-muted);
  }

  .error-wrap {
    padding: 4rem 1rem;
    text-align: center;

    h2 {
      margin-bottom: 0.5rem;
    }

    p {
      color: var(--text-muted);
    }
  }

  .blocked-notice {
    padding: 4rem 1.5rem;
    text-align: center;
    background: var(--surface, rgba(255, 255, 255, 0.02));
    margin: 1.5rem;
    border-radius: 12px;
    border: 1px solid var(--border);

    h2 {
      font-size: 1.3rem;
      margin-bottom: 0.5rem;
      color: var(--text);
    }

    p {
      color: var(--text-muted);
      font-size: 0.95rem;
      max-width: 420px;
      margin: 0 auto;
      line-height: 1.4;
    }
  }

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

  .confirm-card {
    background: var(--bg-card, #1e2029);
    border: 1px solid var(--border);
    border-radius: 14px;
    width: 100%;
    max-width: 400px;
    padding: 24px;
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.5);

    h3 {
      margin: 0 0 10px 0;
      font-size: 1.2rem;
      color: var(--text);
    }

    p {
      margin: 0 0 20px 0;
      font-size: 0.9rem;
      color: var(--text-muted);
      line-height: 1.4;
    }

    .confirm-actions {
      display: flex;
      justify-content: flex-end;
      gap: 10px;

      button {
        padding: 8px 16px;
        border-radius: 8px;
        font-size: 0.9rem;
        font-weight: 600;
        cursor: pointer;
      }

      .btn-cancel {
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text);

        &:hover:not(:disabled) {
          background: rgba(255, 255, 255, 0.05);
        }
      }

      .btn-confirm-block {
        background: #ef4444;
        border: none;
        color: #ffffff;

        &:hover:not(:disabled) {
          opacity: 0.9;
        }

        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      }
    }
  }
</style>
