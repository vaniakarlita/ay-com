<script lang="ts">
  import Link from './Link.svelte';
  import { validateEmail } from './validators';

  let email = $state('');
  let password = $state('');
  let showPassword = $state(false);
  let submitted = $state(false);
  let loading = $state(false);
  let serverError = $state('');

  const emailError = $derived(validateEmail(email));
  const passwordError = $derived(password ? '' : 'Password is required');
  const isValid = $derived(!emailError && !passwordError);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    serverError = '';
    if (!isValid) return;

    loading = true;
    await new Promise((r) => setTimeout(r, 500));
    loading = false;
    serverError = 'Login service is not available yet.';
  }

  function handleGoogle() {
    serverError = 'Google sign-in is not available yet.';
  }
</script>

<main class="page">
  <div class="card">
    <header>
      <Link href="/" class="back">← Landing Page</Link>
      <h1>Sign in to AY.com</h1>
    </header>

    {#if serverError}
      <p class="alert" role="alert">{serverError}</p>
    {/if}

    <button type="button" class="btn btn-outline google" onclick={handleGoogle}>
      Sign in with Google
    </button>

    <p class="divider"><span>or</span></p>

    <form onsubmit={handleSubmit} novalidate>
      <div class="field">
        <label for="email">Email</label>
        <input
          id="email"
          type="email"
          bind:value={email}
          class:invalid={submitted && emailError}
          autocomplete="email"
        />
        {#if submitted && emailError}<p class="error" role="alert">{emailError}</p>{/if}
      </div>

      <div class="field">
        <label for="password">Password</label>
        <div class="password">
          <input
            id="password"
            type={showPassword ? 'text' : 'password'}
            bind:value={password}
            class:invalid={submitted && passwordError}
            autocomplete="current-password"
          />
          <button type="button" class="eye" onclick={() => (showPassword = !showPassword)}>
            {showPassword ? 'Hide' : 'Show'}
          </button>
        </div>
        {#if submitted && passwordError}<p class="error" role="alert">{passwordError}</p>{/if}
      </div>

      <button class="btn btn-primary" type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>

    <p class="footer">
      <Link href="/forgot-password" class="text-link">Forgot password?</Link>
    </p>
    <p class="footer">
      Don't have an account? <Link href="/register" class="text-link">Sign up</Link>
    </p>
  </div>
</main>

<style lang="scss">
  $mobile: 640px;

  .page {
    min-height: 100vh;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding: 4rem 1rem 2rem;

    @media (max-width: $mobile) { padding-top: 3.5rem; }
  }

  .card {
    width: 100%;
    max-width: 440px;
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;

    @media (max-width: $mobile) { padding: 1.25rem; border: 0; }
  }

  h1 { margin: 0.5rem 0 0; font-size: 1.75rem; }

  :global(.back) { font-size: 0.85rem; color: var(--primary); text-decoration: none; }
  :global(.text-link) { color: var(--primary); text-decoration: none; }
  :global(.text-link:hover) { text-decoration: underline; }

  .alert {
    margin: 0;
    padding: 0.75rem 1rem;
    border: 1px solid #f4212e;
    border-radius: 8px;
    color: #f4212e;
    font-size: 0.9rem;
  }

  .google { width: 100%; }

  .divider {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin: 0;
    color: var(--text-muted);
    font-size: 0.85rem;

    &::before, &::after { content: ''; flex: 1; height: 1px; background: var(--border); }
  }

  form { display: flex; flex-direction: column; gap: 1.1rem; }
  .field { display: flex; flex-direction: column; gap: 0.35rem; }
  label { font-size: 0.85rem; font-weight: 700; }

  input {
    width: 100%;
    padding: 0.7rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg);
    color: var(--text);
    font-size: 1rem;

    &:focus { outline: 2px solid var(--primary); }
    &.invalid { border-color: #f4212e; }
  }

  .password { position: relative; }
  .eye {
    position: absolute;
    right: 0.5rem;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: 0;
    color: var(--primary);
    font-weight: 700;
    cursor: pointer;
  }

  .error { margin: 0; font-size: 0.8rem; color: #f4212e; }
  .footer { margin: 0; font-size: 0.9rem; color: var(--text-muted); }
</style>