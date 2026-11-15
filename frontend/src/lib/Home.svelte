<script lang="ts">
  import { onMount } from 'svelte';
  import { authApi, ApiError } from './api';
  import { navigate } from './router';

  let userId = $state('');
  let error = $state('');
  let loggingOut = $state(false);

  onMount(async () => {
    try {
      userId = (await authApi.me()).userId;
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Could not load your profile.';
    }
  });

  async function logout() {
    loggingOut = true;
    try {
      await authApi.logout();
    } finally {
      navigate('/login', true);
    }
  }
</script>

<main style="padding: 2rem">
  <h1>Home</h1>
  {#if error}
    <p role="alert" style="color: #f4212e">{error}</p>
  {:else if userId}
    <p>You are signed in. Your user ID: <strong>{userId}</strong></p>
  {:else}
    <p>Loading...</p>
  {/if}
  <button class="btn btn-outline" onclick={logout} disabled={loggingOut}>
    {loggingOut ? 'Signing out...' : 'Log out'}
  </button>
</main>