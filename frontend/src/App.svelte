<script lang="ts">
  import Bookmarks from './lib/Bookmarks.svelte';
  import ForgotPassword from './lib/ForgotPassword.svelte';
  import Home from './lib/Home.svelte';
  import Landing from './lib/Landing.svelte';
  import Login from './lib/Login.svelte';
  import MemberLayout from './lib/MemberLayout.svelte';
  import Placeholder from './lib/Placeholder.svelte';
  import Register from './lib/Register.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
  import ThreadDetail from './lib/ThreadDetail.svelte';
  import Profile from './lib/Profile.svelte';
  import Settings from './lib/Settings.svelte';
  import { navigate, path } from './lib/router';
  import { isLoggedIn } from './lib/session';

  const memberPages: Record<string, string> = {
    '/home': 'Home',
    '/explore': 'Explore',
    '/notifications': 'Notifications',
    '/messages': 'Messages',
    '/bookmarks': 'Bookmarks',
    '/communities': 'Communities',
    '/premium': 'Premium',
    '/profile': 'Profile',
    '/settings': 'Settings',
  };
  const guestPages = ['/', '/login', '/register', '/forgot-password'];

  const threadMatch = $derived($path.match(/^\/thread\/([a-zA-Z0-9_-]+)$/));
  const isThreadPage = $derived(!!threadMatch);
  const threadId = $derived(threadMatch ? threadMatch[1] : '');

  const profileMatch = $derived($path.match(/^\/profile(?:\/([a-zA-Z0-9_]+))?$/));
  const isProfilePage = $derived(!!profileMatch);
  const profileUsername = $derived(profileMatch && profileMatch[1] ? profileMatch[1] : undefined);

  const isMemberPage = $derived($path in memberPages || isThreadPage || isProfilePage);
  const current = $derived(isMemberPage ? $path : guestPages.includes($path) ? $path : '/');

  $effect(() => {
    if ($isLoggedIn && !isMemberPage) navigate('/home', true);
    else if (!$isLoggedIn && isMemberPage) navigate('/login', true);
  });
</script>

{#if !isMemberPage}
  <ThemeToggle />
{/if}

{#if isMemberPage}
  {#if $isLoggedIn}
    <MemberLayout>
      {#if isThreadPage}
        <ThreadDetail {threadId} />
      {:else if isProfilePage}
        <Profile username={profileUsername} />
      {:else if current === '/home'}
        <Home />
      {:else if current === '/bookmarks'}
        <Bookmarks />
      {:else if current === '/settings'}
        <Settings />
      {:else}
        <Placeholder title={memberPages[current]} />
      {/if}
    </MemberLayout>
  {/if}
{:else if !$isLoggedIn}
  {#if current === '/register'}
    <Register />
  {:else if current === '/login'}
    <Login />
  {:else if current === '/forgot-password'}
    <ForgotPassword />
  {:else}
    <Landing />
  {/if}
{/if}