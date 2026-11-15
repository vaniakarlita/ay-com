<script lang="ts">
  import ForgotPassword from './lib/ForgotPassword.svelte';
  import Home from './lib/Home.svelte';
  import Landing from './lib/Landing.svelte';
  import Login from './lib/Login.svelte';
  import Register from './lib/Register.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
  import { navigate, path } from './lib/router';
  import { isLoggedIn } from './lib/session';

  const known = ['/', '/login', '/register', '/forgot-password', '/home'];

  const current = $derived(known.includes($path) ? $path : '/');
  const isMemberPage = $derived(current === '/home');

  $effect(() => {
    if ($isLoggedIn && !isMemberPage) navigate('/home', true);
    else if (!$isLoggedIn && isMemberPage) navigate('/login', true);
  });
</script>

<ThemeToggle />

{#if isMemberPage}
  {#if $isLoggedIn}<Home />{/if}
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