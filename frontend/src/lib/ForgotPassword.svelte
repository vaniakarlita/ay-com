<script lang="ts">
  import Link from './Link.svelte';
  import { authApi, ApiError } from './api';
  import {
    PASSWORD_RULES,
    validateEmail,
    validatePassword,
    validateConfirm,
  } from './validators';

  type Step = 'email' | 'answer' | 'reset' | 'done';

  let step = $state<Step>('email');
  let email = $state('');
  let question = $state('');
  let answer = $state('');
  let resetToken = $state('');
  let newPassword = $state('');
  let confirm = $state('');
  let showPassword = $state(false);
  let submitted = $state(false);
  let loading = $state(false);
  let error = $state('');

  const emailError = $derived(validateEmail(email));
  const answerError = $derived(answer.trim() ? '' : 'Please answer your security question');
  const passwordError = $derived(validatePassword(newPassword));
  const confirmError = $derived(validateConfirm(newPassword, confirm));

  function reset() {
    submitted = false;
    error = '';
  }

  function message(err: unknown): string {
    return err instanceof ApiError ? err.message : 'Something went wrong. Please try again.';
  }

  async function submitEmail(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    error = '';
    if (emailError || loading) return;

    loading = true;
    try {
      const res = await authApi.forgotQuestion(email.trim().toLowerCase());
      question = res.question;
      step = 'answer';
      reset();
    } catch (err) {
      error = message(err);
    } finally {
      loading = false;
    }
  }

  async function submitAnswer(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    error = '';
    if (answerError || loading) return;

    loading = true;
    try {
      const res = await authApi.forgotVerify(email.trim().toLowerCase(), answer);
      resetToken = res.resetToken;
      step = 'reset';
      reset();
    } catch (err) {
      error = message(err);
    } finally {
      loading = false;
    }
  }

  async function submitPassword(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    error = '';
    if (passwordError || confirmError || loading) return;

    loading = true;
    try {
      await authApi.forgotReset(email.trim().toLowerCase(), resetToken, newPassword);
      step = 'done';
    } catch (err) {
      error = message(err);
    } finally {
      loading = false;
    }
  }
</script>

<main class="page">
  <div class="card">
    <header>
      <Link href="/" class="back">← Landing Page</Link>
      <h1>Forgotten account</h1>
    </header>

    {#if error}
      <p class="alert" role="alert">{error}</p>
    {/if}

    {#if step === 'email'}
      <p class="desc">Enter the email linked to your account.</p>
      <form onsubmit={submitEmail} novalidate>
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
        <button class="btn btn-primary" type="submit" disabled={loading}>
          {loading ? 'Checking...' : 'Next'}
        </button>
      </form>
    {:else if step === 'answer'}
      <p class="desc">Answer your security question to continue.</p>
      <form onsubmit={submitAnswer} novalidate>
        <div class="field">
          <label for="answer">{question}</label>
          <input
            id="answer"
            type="text"
            bind:value={answer}
            class:invalid={submitted && answerError}
            autocomplete="off"
          />
          {#if submitted && answerError}<p class="error" role="alert">{answerError}</p>{/if}
        </div>
        <div class="actions">
          <button type="button" class="btn btn-outline" onclick={() => { step = 'email'; reset(); }}>
            ← Back
          </button>
          <button class="btn btn-primary" type="submit" disabled={loading}>
            {loading ? 'Checking...' : 'Next'}
          </button>
        </div>
      </form>
    {:else if step === 'reset'}
      <p class="desc">Choose a new password.</p>
      <form onsubmit={submitPassword} novalidate>
        <div class="field">
          <label for="new-password">New password</label>
          <div class="password">
            <input
              id="new-password"
              type={showPassword ? 'text' : 'password'}
              bind:value={newPassword}
              class:invalid={submitted && passwordError}
              autocomplete="new-password"
            />
            <button type="button" class="eye" onclick={() => (showPassword = !showPassword)}>
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>
          <ul class="rules">
            {#each PASSWORD_RULES as rule}
              <li class:ok={rule.test(newPassword)}>
                {rule.test(newPassword) ? '✓' : '○'} {rule.label}
              </li>
            {/each}
          </ul>
          {#if submitted && passwordError}<p class="error" role="alert">{passwordError}</p>{/if}
        </div>

        <div class="field">
          <label for="confirm">Confirm new password</label>
          <input
            id="confirm"
            type={showPassword ? 'text' : 'password'}
            bind:value={confirm}
            class:invalid={submitted && confirmError}
            autocomplete="new-password"
          />
          {#if submitted && confirmError}<p class="error" role="alert">{confirmError}</p>{/if}
        </div>

        <button class="btn btn-primary" type="submit" disabled={loading}>
          {loading ? 'Saving...' : 'Reset password'}
        </button>
      </form>
    {:else}
      <p class="success" role="status">Your password has been reset successfully.</p>
      <Link href="/login" class="btn btn-primary">Back to Login</Link>
    {/if}
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
  .desc { margin: 0; font-size: 0.9rem; color: var(--text-muted); }

  .alert {
    margin: 0;
    padding: 0.75rem 1rem;
    border: 1px solid #f4212e;
    border-radius: 8px;
    color: #f4212e;
    font-size: 0.9rem;
  }

  .success {
    margin: 0;
    padding: 0.75rem 1rem;
    border: 1px solid #00ba7c;
    border-radius: 8px;
    color: #00ba7c;
    font-size: 0.9rem;
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

  .rules {
    list-style: none;
    margin: 0;
    padding: 0;
    font-size: 0.78rem;
    color: var(--text-muted);

    li.ok { color: #00ba7c; }
  }

  .error { margin: 0; font-size: 0.8rem; color: #f4212e; }

  .actions { display: flex; gap: 0.75rem; }
  .actions .btn { flex: 1; }
</style>