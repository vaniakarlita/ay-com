<script lang="ts">
  import { onMount } from 'svelte';
  import { navigate } from './router';
  import { userApi, type UserSuggestion } from './api';

  let suggestions = $state<UserSuggestion[]>([]);
  let loading = $state<boolean>(true);
  let followInProgress = $state<Record<string, boolean>>({});

  onMount(async () => {
    try {
      suggestions = await userApi.getSuggestions();
    } catch {
      suggestions = [];
    } finally {
      loading = false;
    }
  });

  async function toggleFollow(user: UserSuggestion) {
    if (followInProgress[user.id]) return;
    followInProgress = { ...followInProgress, [user.id]: true };

    try {
      if (user.isFollowing) {
        await userApi.unfollow(user.id);
        user.isFollowing = false;
        user.followerCount = Math.max(0, user.followerCount - 1);
      } else {
        await userApi.follow(user.id);
        user.isFollowing = true;
        user.followerCount++;
      }
    } catch {
    } finally {
      followInProgress = { ...followInProgress, [user.id]: false };
    }
  }
</script>

<div class="stack">
  <section class="card">
    <h2>What's happening</h2>
    <p class="muted">Trending hashtags will appear here.</p>
  </section>

  <section class="card">
    <h2>Subscribe to Premium</h2>
    <p class="muted">Get a blue checkmark next to your name.</p>
    <button class="btn btn-primary" type="button" onclick={() => navigate('/premium')}>
      Subscribe
    </button>
  </section>

  <section class="card">
    <h2>Who to follow</h2>
    {#if loading}
      <p class="muted">Loading recommendations...</p>
    {:else if suggestions.length === 0}
      <p class="muted">No suggestions available.</p>
    {:else}
      <div class="suggestions-list">
        {#each suggestions as user (user.id)}
          <div class="suggestion-item">
            <button
              type="button"
              class="user-link"
              onclick={() => navigate(`/profile/${user.username}`)}
            >
              <div class="avatar">
                {#if user.profilePictureUrl}
                  <img src={user.profilePictureUrl} alt={user.name} />
                {:else}
                  <div class="avatar-fallback">{user.name.charAt(0).toUpperCase()}</div>
                {/if}
              </div>
              <div class="info">
                <span class="name">{user.name}</span>
                <span class="username">@{user.username}</span>
              </div>
            </button>
            <button
              type="button"
              class="btn-follow"
              class:following={user.isFollowing}
              disabled={followInProgress[user.id]}
              onclick={() => toggleFollow(user)}
            >
              {user.isFollowing ? 'Following' : 'Follow'}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </section>
</div>

<style lang="scss">
  .stack { display: flex; flex-direction: column; gap: 1rem; padding: 1rem; }
  .card {
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  h2 { margin: 0; font-size: 1.1rem; }
  .muted { margin: 0; font-size: 0.9rem; color: var(--text-muted); }
  .btn-primary {
    background-color: var(--accent);
    color: #fff;
    border: none;
    border-radius: 9999px;
    padding: 0.5rem 1rem;
    font-weight: 600;
    cursor: pointer;
  }

  .suggestions-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .suggestion-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .user-link {
    background: none;
    border: none;
    padding: 0;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    cursor: pointer;
    text-align: left;
    color: inherit;
    flex: 1;
    min-width: 0;
  }

  .avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    background-color: var(--border);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  .avatar-fallback {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .info {
    display: flex;
    flex-direction: column;
    min-width: 0;

    .name {
      font-weight: 600;
      font-size: 0.9rem;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .username {
      font-size: 0.8rem;
      color: var(--text-muted);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }

  .btn-follow {
    border-radius: 9999px;
    padding: 0.35rem 0.85rem;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    border: 1px solid var(--accent);
    background-color: var(--accent);
    color: #fff;
    transition: background-color 0.15s ease;

    &:hover {
      opacity: 0.9;
    }

    &.following {
      background-color: transparent;
      color: var(--text);
      border-color: var(--border);

      &:hover {
        background-color: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border-color: #ef4444;
      }
    }
  }
</style>