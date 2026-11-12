<script lang="ts">
  import ForgotPassword from './lib/ForgotPassword.svelte';
  import Landing from './lib/Landing.svelte';
  import Login from './lib/Login.svelte';
  import Register from './lib/Register.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
  import { path } from './lib/router';
</script>

<ThemeToggle />

{#if $path === '/register'}
  <Register />
{:else if $path === '/login'}
  <Login />
{:else if $path === '/forgot-password'}
  <ForgotPassword />
{:else}
  <Landing />
{/if}