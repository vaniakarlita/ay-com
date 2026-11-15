<script lang="ts">
  import Link from './Link.svelte';
  import VerifyStep from './VerifyStep.svelte';
  import { authApi, ApiError, uploadImage } from './api';
  import {
    SECURITY_QUESTIONS, PASSWORD_RULES,
    validateName, validateUsername, validateEmail, validatePassword,
    validateConfirm, validateGender, validateDob, validateImage, validateQuestion,
  } from './validators';

  let step = $state(1);
  let submitted = $state(false);
  let showPassword = $state(false);
  let loading = $state(false);
  let serverError = $state('');

  let form = $state({
    name: '', username: '', email: '', password: '', confirm: '',
    gender: '', dob: '', question: '', answer: '', newsletter: false,
  });
  let profile = $state<File | null>(null);
  let banner = $state<File | null>(null);
  let profilePreview = $state('');
  let bannerPreview = $state('');
  let touched = $state<Record<string, boolean>>({});

  const today = new Date().toISOString().split('T')[0];

  const errors = $derived({
    name: validateName(form.name),
    username: validateUsername(form.username),
    email: validateEmail(form.email),
    password: validatePassword(form.password),
    confirm: validateConfirm(form.password, form.confirm),
    gender: validateGender(form.gender),
    dob: validateDob(form.dob),
    profile: validateImage(profile, 'Profile picture'),
    banner: validateImage(banner, 'Banner'),
    question: validateQuestion(form.question, form.answer),
  });

  const isValid = $derived(Object.values(errors).every((e) => !e));

  function show(field: keyof typeof errors): string {
    return touched[field] || submitted ? errors[field] : '';
  }

  function touch(field: string) {
    touched[field] = true;
  }

  function onFile(e: Event, kind: 'profile' | 'banner') {
    const file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
    touch(kind);
    if (kind === 'profile') {
      if (profilePreview) URL.revokeObjectURL(profilePreview);
      profile = file;
      profilePreview = file ? URL.createObjectURL(file) : '';
    } else {
      if (bannerPreview) URL.revokeObjectURL(bannerPreview);
      banner = file;
      bannerPreview = file ? URL.createObjectURL(file) : '';
    }
  }

  async function handleSubmit(e: SubmitEvent) {
  e.preventDefault();
  submitted = true;
  serverError = '';
  if (!isValid || loading) return;

  loading = true;
  try {
    await authApi.register({
      name: form.name.trim(),
      username: form.username,
      email: form.email.trim().toLowerCase(),
      password: form.password,
      gender: form.gender,
      dateOfBirth: form.dob,
      profilePictureUrl: await uploadImage(profile!, 'profile'),
      bannerUrl: await uploadImage(banner!, 'banner'),
      securityQuestion: form.question,
      securityAnswer: form.answer,
      newsletter: form.newsletter,
    });
    step = 2;
  } catch (err) {
    serverError = err instanceof ApiError ? err.message : 'Registration failed. Please try again.';
  } finally {
    loading = false;
  }
}
</script>

<main class="page">
  <div class="card">
    <header>
      <Link href="/" class="back">← Landing Page</Link>
      <h1>Create your account</h1>
      <p class="step">Step {step} of 2</p>
    </header>

    {#if step === 1}
      {#if serverError}
        <p class="alert" role="alert">{serverError}</p>
      {/if}
      <form onsubmit={handleSubmit} novalidate>
        <div class="field">
          <label for="name">Name</label>
          <input id="name" type="text" bind:value={form.name} onblur={() => touch('name')}
            class:invalid={show('name')} autocomplete="name" />
          {#if show('name')}<p class="error" role="alert">{show('name')}</p>{/if}
        </div>

        <div class="field">
          <label for="username">Username</label>
          <input id="username" type="text" bind:value={form.username} onblur={() => touch('username')}
            class:invalid={show('username')} autocomplete="username" />
          {#if show('username')}<p class="error" role="alert">{show('username')}</p>{/if}
        </div>

        <div class="field">
          <label for="email">Email</label>
          <input id="email" type="email" bind:value={form.email} onblur={() => touch('email')}
            class:invalid={show('email')} autocomplete="email" />
          {#if show('email')}<p class="error" role="alert">{show('email')}</p>{/if}
        </div>

        <div class="field">
          <label for="password">Password</label>
          <div class="password">
            <input id="password" type={showPassword ? 'text' : 'password'}
              bind:value={form.password} onblur={() => touch('password')}
              class:invalid={show('password')} autocomplete="new-password" />
            <button type="button" class="eye" onclick={() => (showPassword = !showPassword)}>
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>
          <ul class="rules">
            {#each PASSWORD_RULES as rule}
              <li class:ok={rule.test(form.password)}>
                {rule.test(form.password) ? '✓' : '○'} {rule.label}
              </li>
            {/each}
          </ul>
        </div>

        <div class="field">
          <label for="confirm">Confirm password</label>
          <input id="confirm" type={showPassword ? 'text' : 'password'}
            bind:value={form.confirm} onblur={() => touch('confirm')}
            class:invalid={show('confirm')} autocomplete="new-password" />
          {#if show('confirm')}<p class="error" role="alert">{show('confirm')}</p>{/if}
        </div>

        <fieldset class="field">
          <legend>Gender</legend>
          <label class="inline"><input type="radio" value="male" bind:group={form.gender} onchange={() => touch('gender')} /> Male</label>
          <label class="inline"><input type="radio" value="female" bind:group={form.gender} onchange={() => touch('gender')} /> Female</label>
          {#if show('gender')}<p class="error" role="alert">{show('gender')}</p>{/if}
        </fieldset>

        <div class="field">
          <label for="dob">Date of birth</label>
          <input id="dob" type="date" max={today} bind:value={form.dob}
            onblur={() => touch('dob')} class:invalid={show('dob')} />
          <p class="hint">This will not be shown publicly.</p>
          {#if show('dob')}<p class="error" role="alert">{show('dob')}</p>{/if}
        </div>

        <div class="field">
          <label for="profile">Profile picture</label>
          <input id="profile" type="file" accept="image/*" onchange={(e) => onFile(e, 'profile')} />
          {#if profilePreview}<img class="avatar" src={profilePreview} alt="Profile preview" />{/if}
          {#if show('profile')}<p class="error" role="alert">{show('profile')}</p>{/if}
        </div>

        <div class="field">
          <label for="banner">Banner</label>
          <input id="banner" type="file" accept="image/*" onchange={(e) => onFile(e, 'banner')} />
          {#if bannerPreview}<img class="banner" src={bannerPreview} alt="Banner preview" />{/if}
          {#if show('banner')}<p class="error" role="alert">{show('banner')}</p>{/if}
        </div>

        <div class="field">
          <label for="question">Personal security question</label>
          <select id="question" bind:value={form.question} onblur={() => touch('question')}
            class:invalid={show('question') && !form.question}>
            <option value="" disabled>Select a question</option>
            {#each SECURITY_QUESTIONS as q}<option value={q}>{q}</option>{/each}
          </select>
          <input type="text" placeholder="Your answer" aria-label="Security answer"
            bind:value={form.answer} onblur={() => touch('question')} />
          {#if show('question')}<p class="error" role="alert">{show('question')}</p>{/if}
        </div>

        <label class="inline">
          <input type="checkbox" bind:checked={form.newsletter} />
          Subscribe to our newsletter
        </label>

        <button class="btn btn-primary" type="submit" disabled={loading || (submitted && !isValid)}>
        {loading ? 'Creating account...' : 'Next'}
        </button>
      </form>
    {:else}
  <VerifyStep email={form.email} onBack={() => (step = 1)} />
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
    max-width: 480px;
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 2rem;

    @media (max-width: $mobile) { padding: 1.25rem; border: 0; }
  }

  h1 { margin: 0.5rem 0 0; font-size: 1.75rem; }
  .step, .hint { margin: 0.25rem 0 0; font-size: 0.8rem; color: var(--text-muted); }

  :global(.back) { font-size: 0.85rem; color: var(--primary); text-decoration: none; }

  form { display: flex; flex-direction: column; gap: 1.1rem; margin-top: 1.5rem; }

  .field { display: flex; flex-direction: column; gap: 0.35rem; border: 0; padding: 0; margin: 0; }
  label, legend { font-size: 0.85rem; font-weight: 700; padding: 0; }
  .inline { display: inline-flex; gap: 0.4rem; align-items: center; font-weight: 400; margin-right: 1rem; }

  input[type='text'], input[type='email'], input[type='password'],
  input[type='date'], select {
    width: 100%;
    padding: 0.7rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg);
    color: var(--text);
    font-size: 1rem;

    &:focus { outline: 2px solid var(--primary); outline-offset: 0; }
    &.invalid { border-color: #f4212e; }
  }

  .password { position: relative; }
  .eye {
    position: absolute; right: 0.5rem; top: 50%; transform: translateY(-50%);
    background: none; border: 0; color: var(--primary); font-weight: 700; cursor: pointer;
  }

  .rules {
    list-style: none; margin: 0; padding: 0; font-size: 0.78rem; color: var(--text-muted);
    li.ok { color: #00ba7c; }
  }

  .error { margin: 0; font-size: 0.8rem; color: #f4212e; }

  .avatar { width: 72px; height: 72px; border-radius: 50%; object-fit: cover; }
  .banner { width: 100%; height: 100px; border-radius: 8px; object-fit: cover; }

    .alert {
    margin: 1rem 0 0;
    padding: 0.75rem 1rem;
    border: 1px solid #f4212e;
    border-radius: 8px;
    color: #f4212e;
    font-size: 0.9rem;
  }
</style>