<script lang="ts">
  import { onMount } from 'svelte';

  type Props = {
    onCredential: (idToken: string) => void;
    text?: 'signin_with' | 'signup_with' | 'continue_with';
  };
  let { onCredential, text = 'continue_with' }: Props = $props();

  const CLIENT_ID: string =
    import.meta.env.VITE_GOOGLE_CLIENT_ID ??
    '138897984454-b4ug0970501074e8m9bmroo5bdfnf79v.apps.googleusercontent.com';

  let host: HTMLDivElement | undefined = $state();
  let failed = $state(false);

  function loadScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      if ((window as any).google?.accounts?.id) return resolve();
      const existing = document.getElementById('google-gsi');
      if (existing) {
        existing.addEventListener('load', () => resolve());
        existing.addEventListener('error', () => reject());
        return;
      }
      const s = document.createElement('script');
      s.id = 'google-gsi';
      s.src = 'https://accounts.google.com/gsi/client';
      s.async = true;
      s.defer = true;
      s.onload = () => resolve();
      s.onerror = () => reject();
      document.head.appendChild(s);
    });
  }

  onMount(async () => {
    try {
      await loadScript();
      if (!host) return;
      const id = (window as any).google.accounts.id;
      id.initialize({
        client_id: CLIENT_ID,
        callback: (res: { credential: string }) => onCredential(res.credential),
      });
      id.renderButton(host, { theme: 'outline', size: 'large', text, width: 320 });
    } catch {
      failed = true;
    }
  });
</script>

<div class="google">
  <div bind:this={host}></div>
  {#if failed}
    <p class="error" role="alert">Google sign-in is unavailable right now.</p>
  {/if}
</div>

<style lang="scss">
  .google { display: flex; justify-content: center; }
  .error { margin: 0; font-size: 0.8rem; color: #f4212e; }
</style>