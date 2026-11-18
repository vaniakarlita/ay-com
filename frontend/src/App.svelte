<script lang="ts">
  import ForgotPassword from './lib/ForgotPassword.svelte';
  import Home from './lib/Home.svelte';
  import Landing from './lib/Landing.svelte';
  import Login from './lib/Login.svelte';
  import MemberLayout from './lib/MemberLayout.svelte';
  import Placeholder from './lib/Placeholder.svelte';
  import Register from './lib/Register.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
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

  const current = $derived($path in memberPages || guestPages.includes($path) ? $path : '/');
  const isMemberPage = $derived(current in memberPages);

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
      {#if current === '/home'}
        <Home />
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