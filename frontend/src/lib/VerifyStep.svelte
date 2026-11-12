<script lang="ts">
  import { onDestroy } from 'svelte';
  import { navigate } from './router';

  let { email, onBack }: { email: string; onBack: () => void } = $props();

  const CODE_LIFETIME = 5 * 60;
  const MOCK_CODE = '123456';

  let code = $state('');
  let error = $state('');
  let info = $state('');
  let secondsLeft = $state(CODE_LIFETIME);
  let loading = $state(false);

  const expired = $derived(secondsLeft <= 0);
  const timeText = $derived(
    `${String(Math.floor(secondsLeft / 60)).padStart(2, '0')}:${String(secondsLeft % 60).padStart(2, '0')}`
  );

  const timer = setInterval(() => {
    if (secondsLeft > 0) secondsLeft--;
  }, 1000);

  onDestroy(() => clearInterval(timer));

  function resend() {
    secondsLeft = CODE_LIFETIME;
    code = '';
    error = '';
    info = 'A new code has been sent to your email.';
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    info = '';
    if (expired) {
      error = 'Verification code has expired. Please request a new one.';
      return;
    }
    if (!/^\d{6}$/.test(code)) {
      error = 'Enter the 6-digit code sent to your email.';
      return;
    }
    loading = true;
    await new Promise((r) => setTimeout(r, 500));
    loading = false;
    if (code !== MOCK_CODE) {
      error = 'Invalid verification code.';
      return;
    }
    error = '';
    navigate('/login');
  }
</script>

<section>
  <h2>We sent you a code</h2>
  <p class="desc">Enter it below to verify <strong>{email}</strong>.</p>

  <form onsubmit={handleSubmit} novalidate>
    <div class="field">
      <label for="code">Verification code</label>
      <input
        id="code"
        type="text"
        inputmode="numeric"
        maxlength="6"
        autocomplete="one-time-code"
        bind:value={code}
        class:invalid={error}
      />
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      {#if info}<p class="info" role="status">{info}</p>{/if}
    </div>

    <p class="timer" class:expired>
      {expired ? 'Code expired' : `Code expires in ${timeText}`}
    </p>

    <button type="button" class="link" onclick={resend}>
      Didn't receive email? Resend code
    </button>

    <div class="actions">
      <button type="button" class="btn btn-outline" onclick={onBack}>← Previous step</button>
      <button type="submit" class="btn btn-primary" disabled={loading}>
        {loading ? 'Verifying...' : 'Next'}
      </button>
    </div>
  </form>
</section>

<style lang="scss">
  h2 { margin: 1rem 0 0.25rem; font-size: 1.4rem; }
  .desc { margin: 0; font-size: 0.85rem; color: var(--text-muted); }

  form { display: flex; flex-direction: column; gap: 1rem; margin-top: 1.25rem; }
  .field { display: flex; flex-direction: column; gap: 0.35rem; }
  label { font-size: 0.85rem; font-weight: 700; }

  input {
    padding: 0.7rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg);
    color: var(--text);
    font-size: 1.2rem;
    letter-spacing: 0.3em;

    &:focus { outline: 2px solid var(--primary); }
    &.invalid { border-color: #f4212e; }
  }

  .error { margin: 0; font-size: 0.8rem; color: #f4212e; }
  .info { margin: 0; font-size: 0.8rem; color: #00ba7c; }
  .timer { margin: 0; font-size: 0.85rem; color: var(--text-muted); &.expired { color: #f4212e; } }

  .link {
    align-self: flex-start;
    background: none; border: 0; padding: 0;
    color: var(--primary); font-size: 0.85rem; cursor: pointer;
    &:hover { text-decoration: underline; }
  }

  .actions { display: flex; gap: 0.75rem; justify-content: space-between; }
  .actions .btn { flex: 1; }
</style>