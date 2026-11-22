<script lang="ts">
  import { onMount } from 'svelte';
  import { userApi, threadApi, type UserProfileResponse, type ThreadItem } from './api';
  import { currentUser } from './user';
  import ThreadCard from './components/ThreadCard.svelte';
  import ThreadSkeleton from './components/ThreadSkeleton.svelte';

  let { username }: { username?: string } = $props();

  let profile = $state<UserProfileResponse | null>(null);
  let threads = $state<ThreadItem[]>([]);
  let activeTab = $state<'posts' | 'replies' | 'likes' | 'media'>('posts');
  let loading = $state(true);
  let threadsLoading = $state(false);
  let errorMsg = $state('');
  let isFollowing = $state(false);
  let followInProgress = $state(false);

  const isOwnProfile = $derived(!username || username === $currentUser?.username);
  const targetUsername = $derived(username || $currentUser?.username || '');

  async function loadProfile() {
    if (!targetUsername) return;
    loading = true;
    errorMsg = '';
    try {
      const data = await userApi.getProfile(targetUsername);
      profile = data;
      isFollowing = data.isFollowing;
      loadUserThreads(data.userId);
    } catch (err: any) {
      errorMsg = err.message || 'User not found';
    } finally {
      loading = false;
    }
  }

  async function loadUserThreads(userId: string) {
    threadsLoading = true;
    try {
      const res = await threadApi.userThreads(userId);
      threads = res.threads ?? [];
    } catch {
      threads = [];
    } finally {
      threadsLoading = false;
    }
  }

  async function toggleFollow() {
    if (!profile || followInProgress) return;
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
              <button class="btn btn-outline" type="button">Edit profile</button>
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

          <div class="stats">
            <span class="stat"><strong>{profile.followingCount}</strong> <span class="label">Following</span></span>
            <span class="stat"><strong>{profile.followerCount}</strong> <span class="label">Followers</span></span>
          </div>
        </div>
      </div>

      <div class="tabs" role="tablist">
        <button
          type="button"
          role="tab"
          class:active={activeTab === 'posts'}
          onclick={() => (activeTab = 'posts')}
        >
          Posts
        </button>
        <button
          type="button"
          role="tab"
          class:active={activeTab === 'replies'}
          onclick={() => (activeTab = 'replies')}
        >
          Replies
        </button>
        {#if isOwnProfile}
          <button
            type="button"
            role="tab"
            class:active={activeTab === 'likes'}
            onclick={() => (activeTab = 'likes')}
          >
            Likes
          </button>
        {/if}
        <button
          type="button"
          role="tab"
          class:active={activeTab === 'media'}
          onclick={() => (activeTab = 'media')}
        >
          Media
        </button>
      </div>
    </header>

    <main class="profile-content">
      {#if activeTab === 'posts'}
        {#if threadsLoading}
          <ThreadSkeleton />
        {:else if threads.length === 0}
          <div class="empty-state">
            <p>No posts yet.</p>
          </div>
        {:else}
          {#each threads as thread (thread.id)}
            <ThreadCard {thread} onDelete={handleThreadDeleted} />
          {/each}
        {/if}
      {:else}
        <div class="empty-state">
          <p>This tab will be available in the next update.</p>
        </div>
      {/if}
    </main>
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

      .stat strong {
        color: var(--text);
      }

      .stat .label {
        color: var(--text-muted);
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
</style>
