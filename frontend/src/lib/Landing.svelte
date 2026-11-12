<script lang="ts">
  import logoLight from '../assets/ay_light.png';
  import logoDark from '../assets/ay_dark.png';
  import { theme } from './theme';
</script>

<main class="landing">
  <section class="logo">
    <img src={$theme === 'dark' ? logoDark : logoLight} alt="AY.com logo" />
  </section>

  <section class="cta">
    <h1>Happening now</h1>
    <h2>Join today.</h2>
    <p class="tagline">Connect, share, engage.</p>

    <p class="label">Don't have an account?</p>
    <a class="btn btn-primary" href="/register">Create account</a>
    <p class="terms">
      By signing up, you agree to the Terms of Service and Privacy Policy,
      including Cookie Use.
    </p>

    <p class="label">Already have an account?</p>
    <a class="btn btn-outline" href="/login">Sign in</a>
  </section>
</main>

<style lang="scss">
  $tablet: 1024px;
  $mobile: 640px;

  .landing {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 1fr 1fr;
    align-items: center;
    gap: 2rem;
    padding: 2rem;

    @media (max-width: $tablet) {
      padding: 1.5rem;
      gap: 1rem;
    }

    @media (max-width: $mobile) {
      grid-template-columns: 1fr;
      justify-items: center;
      text-align: center;
      padding-top: 4rem;
    }
  }

  .logo {
    display: flex;
    justify-content: center;

    img {
      width: min(70%, 380px);

      @media (max-width: $tablet) { width: min(70%, 260px); }
      @media (max-width: $mobile) { width: 120px; }
    }
  }

  .cta {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    width: 100%;
    max-width: 320px;

    @media (max-width: $mobile) { align-items: stretch; }
  }

  h1 {
    margin: 0;
    font-size: clamp(2rem, 5vw, 3.5rem);
  }

  h2 {
    margin: 1rem 0 0;
    font-size: 1.5rem;
  }

  .tagline {
    margin: 0 0 1.5rem;
    font-weight: 600;
  }

  .label {
    margin: 0.5rem 0 0;
    font-weight: 700;
    font-size: 0.9rem;
  }

  .terms {
    margin: 0 0 1.5rem;
    font-size: 0.7rem;
    color: var(--text-muted);
  }

  .btn {
    padding: 0.75rem 1rem;
    border-radius: 999px;
    text-align: center;
    text-decoration: none;
    font-weight: 700;
    transition: background 0.2s;
  }

  .btn-primary {
    background: var(--primary);
    color: #fff;

    &:hover { background: var(--primary-hover); }
  }

  .btn-outline {
    border: 1px solid var(--border);
    color: var(--primary);

    &:hover { background: rgba(29, 155, 240, 0.1); }
  }
</style>