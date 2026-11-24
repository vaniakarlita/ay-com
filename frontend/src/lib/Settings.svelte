<script lang="ts">
  import { onMount } from 'svelte';
  import { userApi, type FollowUserItem } from './api';
  import { currentUser } from './user';
  import { clearSession } from './session';
  import { navigate } from './router';
  import { fontSize, fontColor, type FontSize, type FontColor } from './display';

  type SettingsTab = 'blocked' | 'security' | 'account' | 'display' | 'notifications';

  let activeTab = $state<SettingsTab>('blocked');

  // Blocked Accounts State
  let blockedUsers = $state<FollowUserItem[]>([]);
  let blockedLoading = $state(false);
  let blockedActionLoading = $state<Record<string, boolean>>({});

  // Security (Private Account) State
  let isPrivate = $state(false);
  let savingPrivacy = $state(false);
  let privacySuccess = $state('');

  // Deactivate State
  let deactivatePassword = $state('');
  let deactivating = $state(false);
  let deactivateError = $state('');
  let showDeactivateModal = $state(false);

  // Notification Preferences State
  let notifLike = $state(true);
  let notifRepost = $state(true);
  let notifFollow = $state(true);
  let notifMention = $state(true);
  let notifCommunity = $state(true);
  let notifLoading = $state(false);
  let notifSavedMsg = $state('');

  onMount(async () => {
    if ($currentUser) {
      isPrivate = !!$currentUser.isPrivate;
    }
    loadBlockedAccounts();
    loadNotificationSettings();
  });

  async function loadBlockedAccounts() {
    blockedLoading = true;
    try {
      const res = await userApi.getBlockedUsers('', 50);
      blockedUsers = res.users || [];
    } catch {
      blockedUsers = [];
    } finally {
      blockedLoading = false;
    }
  }

  async function handleUnblock(userId: string) {
    blockedActionLoading[userId] = true;
    try {
      await userApi.unblock(userId);
      blockedUsers = blockedUsers.filter((u) => u.userId !== userId);
    } catch {
    } finally {
      blockedActionLoading[userId] = false;
    }
  }

  async function handlePrivacyToggle() {
    savingPrivacy = true;
    privacySuccess = '';
    const nextVal = !isPrivate;
    try {
      const res = await userApi.updateProfile({ isPrivate: nextVal });
      isPrivate = nextVal;
      if ($currentUser) {
        $currentUser.isPrivate = nextVal;
      }
      privacySuccess = `Your account is now ${nextVal ? 'private' : 'public'}.`;
      setTimeout(() => (privacySuccess = ''), 3000);
    } catch {
    } finally {
      savingPrivacy = false;
    }
  }

  async function handleDeactivateSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!deactivatePassword) return;
    deactivating = true;
    deactivateError = '';
    try {
      await userApi.deactivate(deactivatePassword);
      clearSession();
      currentUser.set(null);
      navigate('/login', true);
    } catch (err: any) {
      deactivateError = err.message || 'Failed to deactivate account';
    } finally {
      deactivating = false;
    }
  }

  async function loadNotificationSettings() {
    notifLoading = true;
    try {
      const res = await userApi.getNotificationSettings();
      notifLike = res.notifLike;
      notifRepost = res.notifRepost;
      notifFollow = res.notifFollow;
      notifMention = res.notifMention;
      notifCommunity = res.notifCommunity;
    } catch {
    } finally {
      notifLoading = false;
    }
  }

  async function saveNotificationSettings() {
    notifSavedMsg = '';
    try {
      await userApi.updateNotificationSettings({
        notifLike,
        notifRepost,
        notifFollow,
        notifMention,
        notifCommunity,
      });
      notifSavedMsg = 'Preferences saved successfully.';
      setTimeout(() => (notifSavedMsg = ''), 3000);
    } catch {
    }
  }
</script>

<div class="settings-container">
  <div class="settings-sidebar">
    <div class="sidebar-header">
      <h2>Settings</h2>
    </div>
    <nav class="settings-nav">
      <button
        type="button"
        class="nav-item"
        class:active={activeTab === 'blocked'}
        onclick={() => (activeTab = 'blocked')}
      >
        Blocked Accounts
      </button>
      <button
        type="button"
        class="nav-item"
        class:active={activeTab === 'security'}
        onclick={() => (activeTab = 'security')}
      >
        Security
      </button>
      <button
        type="button"
        class="nav-item"
        class:active={activeTab === 'account'}
        onclick={() => (activeTab = 'account')}
      >
        Your Account
      </button>
      <button
        type="button"
        class="nav-item"
        class:active={activeTab === 'display'}
        onclick={() => (activeTab = 'display')}
      >
        Display
      </button>
      <button
        type="button"
        class="nav-item"
        class:active={activeTab === 'notifications'}
        onclick={() => (activeTab = 'notifications')}
      >
        Notification Preferences
      </button>
    </nav>
  </div>

  <main class="settings-content">
    {#if activeTab === 'blocked'}
      <section class="section">
        <h2 class="section-title">Blocked Accounts</h2>
        <p class="section-desc">
          When you block someone, that person won't be able to follow you or view your posts, and mutual follows are removed.
        </p>

        {#if blockedLoading}
          <p class="loading-state">Loading blocked accounts...</p>
        {:else if blockedUsers.length === 0}
          <div class="empty-state">
            <p>You haven't blocked any accounts.</p>
          </div>
        {:else}
          <ul class="blocked-list">
            {#each blockedUsers as u (u.userId)}
              <li class="user-row">
                <div class="user-info">
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
                <button
                  type="button"
                  class="btn-unblock"
                  disabled={blockedActionLoading[u.userId]}
                  onclick={() => handleUnblock(u.userId)}
                >
                  {blockedActionLoading[u.userId] ? 'Unblocking...' : 'Unblock'}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </section>
    {:else if activeTab === 'security'}
      <section class="section">
        <h2 class="section-title">Security & Privacy</h2>
        <p class="section-desc">Manage how visible your profile and threads are to other users on AY.com.</p>

        {#if privacySuccess}
          <div class="alert alert-success">{privacySuccess}</div>
        {/if}

        <div class="setting-row">
          <div class="row-info">
            <span class="row-title">Private Account</span>
            <span class="row-desc">
              When your account is private, only your followers can see your posts, replies, media, and follow lists.
            </span>
          </div>
          <button
            type="button"
            class="toggle-switch"
            class:active={isPrivate}
            disabled={savingPrivacy}
            onclick={handlePrivacyToggle}
            aria-label="Toggle private account"
          >
            <div class="toggle-slider"></div>
          </button>
        </div>
      </section>
    {:else if activeTab === 'account'}
      <section class="section">
        <h2 class="section-title">Your Account</h2>
        <p class="section-desc">Deactivate your account and permanently remove your access to AY.com.</p>

        <div class="danger-zone">
          <div class="danger-header">
            <h3>Deactivate Account</h3>
            <p>
              This will deactivate your account and log you out. Your profile and posts will no longer be visible on AY.com, and you will not be able to log in again.
            </p>
          </div>
          <button
            type="button"
            class="btn-danger"
            onclick={() => {
              showDeactivateModal = true;
              deactivateError = '';
              deactivatePassword = '';
            }}
          >
            Deactivate account
          </button>
        </div>
      </section>
    {:else if activeTab === 'display'}
      <section class="section">
        <h2 class="section-title">Display Settings</h2>
        <p class="section-desc">Customize your font size and reading colors for optimal viewing experience.</p>

        <div class="setting-group">
          <h3 class="group-title">Font Size</h3>
          <div class="radio-cards">
            {#each (['small', 'normal', 'large'] as FontSize[]) as s}
              <label class="radio-card" class:selected={$fontSize === s}>
                <input
                  type="radio"
                  name="font-size"
                  value={s}
                  checked={$fontSize === s}
                  onchange={() => ($fontSize = s)}
                />
                <span class="radio-label">{s.charAt(0).toUpperCase() + s.slice(1)}</span>
              </label>
            {/each}
          </div>
        </div>

        <div class="setting-group">
          <h3 class="group-title">Font Color</h3>
          <div class="radio-cards">
            {#each ([
              { id: 'default', label: 'Default' },
              { id: 'warm', label: 'Warm Sepia' },
              { id: 'cool', label: 'Cool Slate' },
              { id: 'contrast', label: 'High Contrast' },
            ] as { id: FontColor; label: string }[]) as c}
              <label class="radio-card" class:selected={$fontColor === c.id}>
                <input
                  type="radio"
                  name="font-color"
                  value={c.id}
                  checked={$fontColor === c.id}
                  onchange={() => ($fontColor = c.id)}
                />
                <span class="radio-label">{c.label}</span>
              </label>
            {/each}
          </div>
        </div>
      </section>
    {:else if activeTab === 'notifications'}
      <section class="section">
        <h2 class="section-title">Notification Preferences</h2>
        <p class="section-desc">Select what activities trigger push and in-app notifications.</p>

        {#if notifSavedMsg}
          <div class="alert alert-success">{notifSavedMsg}</div>
        {/if}

        <div class="setting-list">
          <div class="setting-row">
            <div class="row-info">
              <span class="row-title">Likes</span>
              <span class="row-desc">Notify when someone likes your thread or reply</span>
            </div>
            <button
              type="button"
              class="toggle-switch"
              class:active={notifLike}
              onclick={() => {
                notifLike = !notifLike;
                saveNotificationSettings();
              }}
              aria-label="Toggle like notifications"
            >
              <div class="toggle-slider"></div>
            </button>
          </div>

          <div class="setting-row">
            <div class="row-info">
              <span class="row-title">Reposts</span>
              <span class="row-desc">Notify when someone reposts your thread</span>
            </div>
            <button
              type="button"
              class="toggle-switch"
              class:active={notifRepost}
              onclick={() => {
                notifRepost = !notifRepost;
                saveNotificationSettings();
              }}
              aria-label="Toggle repost notifications"
            >
              <div class="toggle-slider"></div>
            </button>
          </div>

          <div class="setting-row">
            <div class="row-info">
              <span class="row-title">Follows</span>
              <span class="row-desc">Notify when someone starts following you</span>
            </div>
            <button
              type="button"
              class="toggle-switch"
              class:active={notifFollow}
              onclick={() => {
                notifFollow = !notifFollow;
                saveNotificationSettings();
              }}
              aria-label="Toggle follow notifications"
            >
              <div class="toggle-slider"></div>
            </button>
          </div>

          <div class="setting-row">
            <div class="row-info">
              <span class="row-title">Mentions</span>
              <span class="row-desc">Notify when someone mentions @{ $currentUser?.username || 'you' } in a thread</span>
            </div>
            <button
              type="button"
              class="toggle-switch"
              class:active={notifMention}
              onclick={() => {
                notifMention = !notifMention;
                saveNotificationSettings();
              }}
              aria-label="Toggle mention notifications"
            >
              <div class="toggle-slider"></div>
            </button>
          </div>

          <div class="setting-row">
            <div class="row-info">
              <span class="row-title">Community</span>
              <span class="row-desc">Notify on activities in communities you have joined</span>
            </div>
            <button
              type="button"
              class="toggle-switch"
              class:active={notifCommunity}
              onclick={() => {
                notifCommunity = !notifCommunity;
                saveNotificationSettings();
              }}
              aria-label="Toggle community notifications"
            >
              <div class="toggle-slider"></div>
            </button>
          </div>
        </div>
      </section>
    {/if}
  </main>
</div>

{#if showDeactivateModal}
  <div
    class="modal-backdrop"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget && !deactivating) showDeactivateModal = false;
    }}
  >
    <div class="confirm-card" role="dialog" aria-modal="true">
      <h3>Confirm Deactivation</h3>
      <p>
        Are you sure you want to deactivate your account? This action is permanent and cannot be undone. Enter your current password to proceed.
      </p>

      {#if deactivateError}
        <div class="alert alert-error">{deactivateError}</div>
      {/if}

      <form onsubmit={handleDeactivateSubmit}>
        <div class="field-group">
          <label for="deactivate-pass">Current Password</label>
          <input
            id="deactivate-pass"
            type="password"
            bind:value={deactivatePassword}
            placeholder="Enter password"
            required
            disabled={deactivating}
          />
        </div>

        <div class="modal-actions">
          <button
            type="button"
            class="btn-secondary"
            disabled={deactivating}
            onclick={() => (showDeactivateModal = false)}
          >
            Cancel
          </button>
          <button type="submit" class="btn-danger-confirm" disabled={deactivating || !deactivatePassword}>
            {deactivating ? 'Deactivating...' : 'Yes, Deactivate'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style lang="scss">
  .settings-container {
    display: flex;
    min-height: 100vh;
    border-right: 1px solid var(--border);

    @media (max-width: 768px) {
      flex-direction: column;
    }
  }

  .settings-sidebar {
    width: 260px;
    border-right: 1px solid var(--border);
    flex-shrink: 0;

    @media (max-width: 768px) {
      width: 100%;
      border-right: none;
      border-bottom: 1px solid var(--border);
    }

    .sidebar-header {
      padding: 1.25rem 1rem;
      border-bottom: 1px solid var(--border);

      h2 {
        margin: 0;
        font-size: 1.25rem;
        font-weight: 800;
      }
    }

    .settings-nav {
      display: flex;
      flex-direction: column;

      .nav-item {
        background: transparent;
        border: none;
        padding: 1rem;
        text-align: left;
        font-size: 0.95rem;
        font-weight: 600;
        color: var(--text-muted);
        cursor: pointer;
        transition: background 0.15s, color 0.15s;

        &:hover {
          background: rgba(255, 255, 255, 0.04);
          color: var(--text);
        }

        &.active {
          color: var(--text);
          background: rgba(255, 255, 255, 0.08);
          border-right: 3px solid var(--primary);
        }
      }
    }
  }

  .settings-content {
    flex: 1;
    padding: 1.5rem 2rem;
    max-width: 680px;

    @media (max-width: 768px) {
      padding: 1rem;
    }

    .section {
      .section-title {
        margin: 0 0 0.5rem 0;
        font-size: 1.35rem;
        font-weight: 800;
      }

      .section-desc {
        color: var(--text-muted);
        margin: 0 0 1.5rem 0;
        line-height: 1.4;
      }
    }
  }

  .setting-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 0;
    border-bottom: 1px solid var(--border);

    .row-info {
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
      padding-right: 1.5rem;

      .row-title {
        font-weight: 700;
        font-size: 1rem;
      }

      .row-desc {
        font-size: 0.85rem;
        color: var(--text-muted);
        line-height: 1.35;
      }
    }
  }

  .toggle-switch {
    width: 48px;
    height: 26px;
    background: var(--border);
    border-radius: 999px;
    border: none;
    cursor: pointer;
    position: relative;
    padding: 2px;
    transition: background 0.2s;
    flex-shrink: 0;

    .toggle-slider {
      width: 22px;
      height: 22px;
      background: #fff;
      border-radius: 50%;
      transition: transform 0.2s;
    }

    &.active {
      background: var(--primary);

      .toggle-slider {
        transform: translateX(22px);
      }
    }
  }

  .blocked-list {
    list-style: none;
    padding: 0;
    margin: 0;

    .user-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0.85rem 0;
      border-bottom: 1px solid var(--border);

      .user-info {
        display: flex;
        align-items: center;
        gap: 0.75rem;

        .avatar {
          width: 40px;
          height: 40px;
          border-radius: 50%;
          object-fit: cover;
        }

        .avatar-fallback {
          width: 40px;
          height: 40px;
          border-radius: 50%;
          background: rgba(255, 255, 255, 0.1);
          display: flex;
          align-items: center;
          justify-content: center;
          font-weight: 700;
        }

        .names {
          display: flex;
          flex-direction: column;

          .display-name {
            font-weight: 700;
            font-size: 0.95rem;
          }

          .user-handle {
            color: var(--text-muted);
            font-size: 0.85rem;
          }
        }
      }

      .btn-unblock {
        padding: 0.45rem 1rem;
        border-radius: 999px;
        background: transparent;
        border: 1px solid #ef4444;
        color: #ef4444;
        font-weight: 700;
        font-size: 0.85rem;
        cursor: pointer;
        transition: all 0.15s;

        &:hover:not(:disabled) {
          background: rgba(239, 68, 68, 0.1);
        }
      }
    }
  }

  .danger-zone {
    border: 1px solid rgba(239, 68, 68, 0.3);
    background: rgba(239, 68, 68, 0.04);
    border-radius: 12px;
    padding: 1.5rem;

    .danger-header {
      h3 {
        margin: 0 0 0.5rem 0;
        color: #ef4444;
      }
      p {
        margin: 0 0 1rem 0;
        font-size: 0.9rem;
        color: var(--text-muted);
        line-height: 1.4;
      }
    }

    .btn-danger {
      padding: 0.6rem 1.25rem;
      border-radius: 999px;
      background: #ef4444;
      border: none;
      color: #fff;
      font-weight: 700;
      cursor: pointer;

      &:hover {
        opacity: 0.9;
      }
    }
  }

  .setting-group {
    margin-bottom: 2rem;

    .group-title {
      font-size: 1.05rem;
      font-weight: 700;
      margin-bottom: 0.75rem;
    }

    .radio-cards {
      display: flex;
      flex-wrap: wrap;
      gap: 0.75rem;

      .radio-card {
        flex: 1;
        min-width: 120px;
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.85rem 1rem;
        border: 1px solid var(--border);
        border-radius: 10px;
        cursor: pointer;
        transition: border-color 0.15s, background 0.15s;

        input[type='radio'] {
          accent-color: var(--primary);
        }

        &.selected {
          border-color: var(--primary);
          background: rgba(29, 155, 240, 0.08);
        }

        .radio-label {
          font-weight: 600;
          font-size: 0.95rem;
        }
      }
    }
  }

  .alert {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    font-size: 0.9rem;
    margin-bottom: 1rem;

    &.alert-success {
      background: rgba(34, 197, 94, 0.15);
      border: 1px solid rgba(34, 197, 94, 0.3);
      color: #22c55e;
    }

    &.alert-error {
      background: rgba(239, 68, 68, 0.15);
      border: 1px solid rgba(239, 68, 68, 0.3);
      color: #ef4444;
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
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 14px;
    width: 100%;
    max-width: 440px;
    padding: 24px;
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.5);

    h3 {
      margin: 0 0 10px 0;
      color: #ef4444;
      font-size: 1.25rem;
    }

    p {
      margin: 0 0 20px 0;
      font-size: 0.9rem;
      color: var(--text-muted);
      line-height: 1.4;
    }

    .field-group {
      display: flex;
      flex-direction: column;
      gap: 6px;
      margin-bottom: 1.5rem;

      label {
        font-weight: 600;
        font-size: 0.85rem;
      }

      input {
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 10px;
        color: var(--text);
        font-size: 0.95rem;

        &:focus {
          outline: none;
          border-color: #ef4444;
        }
      }
    }

    .modal-actions {
      display: flex;
      justify-content: flex-end;
      gap: 12px;

      button {
        padding: 8px 16px;
        border-radius: 8px;
        font-size: 0.9rem;
        font-weight: 700;
        cursor: pointer;
      }

      .btn-secondary {
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text);
      }

      .btn-danger-confirm {
        background: #ef4444;
        border: none;
        color: #fff;

        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      }
    }
  }

  .empty-state,
  .loading-state {
    padding: 2rem 0;
    text-align: center;
    color: var(--text-muted);
  }
</style>
