<script lang="ts">
  import { onMount, type Snippet } from 'svelte';
  import { authApi } from './api';
  import RightSidebar from './RightSidebar.svelte';
  import { navigate, path } from './router';
  import { theme, toggleTheme } from './theme';
  import { currentUser, loadCurrentUser } from './user';

  let { children }: { children?: Snippet } = $props();

  const items = [
    { href: '/home', label: 'Home', icon: 'M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z M9 22V12h6v10' },
    { href: '/explore', label: 'Explore', icon: 'M3 11a8 8 0 1 0 16 0a8 8 0 1 0-16 0 M21 21l-4.35-4.35' },
    { href: '/notifications', label: 'Notifications', icon: 'M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9 M13.73 21a2 2 0 0 1-3.46 0' },
    { href: '/messages', label: 'Messages', icon: 'M4 4h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z M22 6l-10 7L2 6' },
    { href: '/bookmarks', label: 'Bookmarks', icon: 'M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z' },
    { href: '/communities', label: 'Communities', icon: 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M5 7a4 4 0 1 0 8 0a4 4 0 1 0-8 0 M23 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75' },
    { href: '/premium', label: 'Premium', icon: 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01z' },
    { href: '/profile', label: 'Profile', icon: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M8 7a4 4 0 1 0 8 0a4 4 0 1 0-8 0' },
    { href: '/settings', label: 'Settings', icon: 'M4 21v-7 M4 10V3 M12 21v-9 M12 8V3 M20 21v-5 M20 12V3 M1 14h6 M9 8h6 M17 16h6' },
  ];

  let menuOpen = $state(false);
  let loggingOut = $state(false);

  const displayName = $derived($currentUser?.name ?? 'Your account');
  const handle = $derived($currentUser?.username ? `@${$currentUser.username}` : '');
  const initial = $derived(($currentUser?.name ?? '?').charAt(0).toUpperCase());

  onMount(() => {
    loadCurrentUser();
  });

  function go(e: MouseEvent, href: string) {
    e.preventDefault();
    menuOpen = false;
    navigate(href);
  }

  async function logout() {
    loggingOut = true;
    try {
      await authApi.logout();
    } finally {
      currentUser.set(null);
      navigate('/login', true);
    }
  }
</script>

<div class="shell">
  <aside class="left">
    <nav aria-label="Main">
      <a class="brand" href="/home" onclick={(e) => go(e, '/home')}>AY.com</a>

      {#each items as item}
        <a
          class="item"
          class:active={$path === item.href}
          class:extra={item.href === '/bookmarks' || item.href === '/communities' || item.href === '/premium' || item.href === '/settings'}
          href={item.href}
          aria-label={item.label}
          onclick={(e) => go(e, item.href)}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d={item.icon} /></svg>
          <span class="label">{item.label}</span>
        </a>
      {/each}

      <button class="post" type="button">
        <span class="long">Post</span>
        <span class="short">+</span>
      </button>
    </nav>

    <div class="bottom">
      <button class="theme" type="button" onclick={toggleTheme}>
        <span class="label">{$theme === 'light' ? 'Dark mode' : 'Light mode'}</span>
        <span class="short">{$theme === 'light' ? '☾' : '☀'}</span>
      </button>

      <div class="account">
        {#if menuOpen}
          <div class="menu" role="menu">
            <button type="button" role="menuitem" onclick={toggleTheme}>
              Switch to {$theme === 'light' ? 'dark' : 'light'} theme
            </button>
            <button type="button" role="menuitem" onclick={logout} disabled={loggingOut}>
              {loggingOut ? 'Signing out...' : `Log out ${handle}`}
            </button>
          </div>
        {/if}

        <button
          class="me"
          type="button"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onclick={() => (menuOpen = !menuOpen)}
        >
          {#if $currentUser?.profilePictureUrl}
            <img class="avatar" src={$currentUser.profilePictureUrl} alt="" />
          {:else}
            <span class="avatar fallback">{initial}</span>
          {/if}
          <span class="who">
            <strong>
              {displayName}
              {#if $currentUser?.isVerified}<span class="verified" aria-label="Verified">✓</span>{/if}
            </strong>
            <small>{handle}</small>
          </span>
        </button>
      </div>
    </div>
  </aside>

  <main class="center">
    {@render children?.()}
  </main>

  <aside class="right">
    <RightSidebar />
  </aside>
</div>

<style lang="scss">
  $tablet: 1024px;
  $mobile: 640px;

  .shell {
    display: grid;
    grid-template-columns: 275px minmax(0, 600px) 350px;
    justify-content: center;
    height: 100dvh;
    color: var(--text);

    @media (max-width: $tablet) { grid-template-columns: 80px minmax(0, 600px); }
    @media (max-width: $mobile) { grid-template-columns: minmax(0, 1fr); }
  }

  .left, .center, .right { height: 100dvh; overflow-y: auto; }

  .center {
    border-left: 1px solid var(--border);
    border-right: 1px solid var(--border);

    @media (max-width: $mobile) { border: 0; padding-bottom: 4.5rem; }
  }

  .right { @media (max-width: $tablet) { display: none; } }

  .left {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    padding: 0.5rem 0.75rem 1rem;

    @media (max-width: $tablet) { align-items: center; padding-inline: 0.5rem; }

    @media (max-width: $mobile) {
      position: fixed;
      inset: auto 0 0 0;
      height: auto;
      flex-direction: row;
      padding: 0;
      background: var(--bg);
      border-top: 1px solid var(--border);
      z-index: 20;
      overflow: visible;
    }
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;

    @media (max-width: $tablet) { align-items: center; }
    @media (max-width: $mobile) { flex-direction: row; justify-content: space-around; width: 100%; }
  }

  .brand {
    padding: 0.75rem;
    font-size: 1.3rem;
    font-weight: 800;
    color: var(--text);
    text-decoration: none;

    @media (max-width: $tablet) { font-size: 0.9rem; padding: 0.75rem 0; }
    @media (max-width: $mobile) { display: none; }
  }

  .item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem;
    border-radius: 999px;
    color: var(--text);
    text-decoration: none;
    font-size: 1.15rem;

    &:hover { background: var(--border); }
    &.active { font-weight: 800; color: var(--primary); }

    @media (max-width: $mobile) {
      padding: 0.9rem;
      &.extra { display: none; }
    }
  }

  svg {
    width: 26px;
    height: 26px;
    flex: none;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .label { @media (max-width: $tablet) { display: none; } }

  .post {
    margin-top: 0.75rem;
    padding: 0.85rem 1rem;
    border: 0;
    border-radius: 999px;
    background: var(--primary);
    color: #fff;
    font-size: 1.05rem;
    font-weight: 800;
    cursor: pointer;

    .short { display: none; }

    @media (max-width: $tablet) {
      width: 48px;
      height: 48px;
      padding: 0;
      .long { display: none; }
      .short { display: inline; font-size: 1.5rem; }
    }

    @media (max-width: $mobile) { display: none; }
  }

  .bottom {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;

    @media (max-width: $tablet) { align-items: center; }

    @media (max-width: $mobile) {
      position: fixed;
      top: 0.5rem;
      right: 0.5rem;
      z-index: 30;
    }
  }

  .theme {
    padding: 0.6rem 1rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--bg);
    color: var(--text);
    font-weight: 700;
    cursor: pointer;

    &:hover { background: var(--border); }
    .short { display: none; }

    @media (max-width: $tablet) {
      padding: 0.5rem 0.7rem;
      .short { display: inline; }
    }

    @media (max-width: $mobile) { display: none; }
  }

  .account { position: relative; }

  .me {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem;
    border: 0;
    border-radius: 999px;
    background: none;
    color: var(--text);
    text-align: left;
    cursor: pointer;

    &:hover { background: var(--border); }
  }

  .avatar {
    width: 40px;
    height: 40px;
    flex: none;
    border-radius: 50%;
    object-fit: cover;

    &.fallback {
      display: grid;
      place-items: center;
      background: var(--primary);
      color: #fff;
      font-weight: 800;
    }
  }

  .who {
    display: flex;
    flex-direction: column;
    min-width: 0;
    line-height: 1.2;

    strong, small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    small { color: var(--text-muted); }

    @media (max-width: $tablet) { display: none; }
  }

  .verified { color: #1d9bf0; margin-left: 0.2rem; }

  .menu {
    position: absolute;
    bottom: calc(100% + 0.5rem);
    left: 0;
    min-width: 240px;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg);
    box-shadow: 0 4px 24px rgb(0 0 0 / 0.25);
    overflow: hidden;

    button {
      padding: 0.85rem 1rem;
      border: 0;
      background: none;
      color: var(--text);
      font-weight: 700;
      text-align: left;
      cursor: pointer;

      &:hover { background: var(--border); }
    }

    @media (max-width: $tablet) { left: 0; }
    @media (max-width: $mobile) { bottom: auto; top: calc(100% + 0.5rem); left: auto; right: 0; }
  }
</style>