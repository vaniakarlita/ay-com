<script lang="ts">
  import { onMount } from 'svelte';
  import { threadApi, type CategoryItem, type ThreadItem } from '../api';
  import { currentUser } from '../user';
  import WordCounter from './WordCounter.svelte';

  let {
    onCreated,
    placeholder = "What's happening?",
    autoFocus = false,
  }: {
    onCreated?: (thread: ThreadItem) => void;
    placeholder?: string;
    autoFocus?: boolean;
  } = $props();

  let content = $state('');
  let replyPermission = $state('everyone');
  let selectedCategory = $state('');
  let isScheduling = $state(false);
  let scheduledDate = $state('');
  let scheduledTime = $state('');
  let isPosting = $state(false);
  let errorMsg = $state('');
  let categories = $state<CategoryItem[]>([]);

  // Calculate word count (splitting by whitespace)
  const wordCount = $derived.by(() => {
    const trimmed = content.trim();
    if (!trimmed) return 0;
    return trimmed.split(/\s+/).length;
  });

  const maxWords = 100; // Limit word count for circular indicator demonstration
  const isPostDisabled = $derived(!content.trim() || wordCount > maxWords || isPosting);
  const initial = $derived(($currentUser?.name ?? 'U').charAt(0).toUpperCase());

  onMount(async () => {
    try {
      categories = await threadApi.categories();
    } catch {
      // Fallback default categories
      categories = [
        { id: '1', name: 'Technology' },
        { id: '2', name: 'Gaming' },
        { id: '3', name: 'Sports' },
        { id: '4', name: 'Entertainment' },
      ];
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (isPostDisabled) return;

    isPosting = true;
    errorMsg = '';

    let scheduledAt = '';
    if (isScheduling && scheduledDate) {
      const timeVal = scheduledTime || '12:00';
      scheduledAt = new Date(`${scheduledDate}T${timeVal}`).toISOString();
    }

    try {
      const thread = await threadApi.create({
        content: content.trim(),
        replyPermission,
        categories: selectedCategory ? [selectedCategory] : [],
        scheduledAt: scheduledAt || undefined,
      });

      content = '';
      selectedCategory = '';
      isScheduling = false;
      scheduledDate = '';
      scheduledTime = '';
      onCreated?.(thread);
    } catch (err: any) {
      errorMsg = err.message || 'Failed to post thread';
    } finally {
      isPosting = false;
    }
  }
</script>

<form class="create-thread-form" onsubmit={handleSubmit}>
  {#if errorMsg}
    <div class="error-banner">{errorMsg}</div>
  {/if}

  <div class="input-row">
    <!-- User Avatar -->
    <div class="avatar-wrap">
      {#if $currentUser?.profilePictureUrl}
        <img src={$currentUser.profilePictureUrl} alt="" class="avatar" />
      {:else}
        <div class="avatar fallback">{initial}</div>
      {/if}
    </div>

    <!-- Textarea -->
    <div class="textarea-wrap">
      <textarea
        bind:value={content}
        {placeholder}
        rows="3"
        maxlength="1000"
      ></textarea>

      <!-- Options Row (Categories & Permissions) -->
      <div class="options-bar">
        <!-- Category selector -->
        <div class="select-group">
          <label for="cat-select" class="sr-only">Category</label>
          <select id="cat-select" bind:value={selectedCategory}>
            <option value="">No Category</option>
            {#each categories as cat}
              <option value={cat.name}>{cat.name}</option>
            {/each}
          </select>
        </div>

        <!-- Reply Permission selector -->
        <div class="select-group">
          <label for="perm-select" class="sr-only">Who can reply</label>
          <select id="perm-select" bind:value={replyPermission}>
            <option value="everyone">Everyone can reply</option>
            <option value="following">Accounts you follow</option>
            <option value="verified">Verified accounts only</option>
          </select>
        </div>

        <!-- Schedule toggle button -->
        <button
          type="button"
          class="schedule-btn"
          class:active={isScheduling}
          onclick={() => (isScheduling = !isScheduling)}
          title="Schedule thread"
        >
          <svg viewBox="0 0 24 24" class="svg-icon">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>Schedule</span>
        </button>
      </div>

      <!-- Schedule inputs (if active) -->
      {#if isScheduling}
        <div class="schedule-inputs">
          <label>
            Date:
            <input type="date" bind:value={scheduledDate} min={new Date().toISOString().split('T')[0]} />
          </label>
          <label>
            Time:
            <input type="time" bind:value={scheduledTime} />
          </label>
        </div>
      {/if}

      <!-- Bottom controls (Counter + Submit) -->
      <div class="bottom-bar">
        <div class="counter-wrap">
          <WordCounter count={wordCount} max={maxWords} />
        </div>

        <button type="submit" class="submit-btn" disabled={isPostDisabled}>
          {isPosting ? 'Posting...' : isScheduling ? 'Schedule' : 'Post'}
        </button>
      </div>
    </div>
  </div>
</form>

<style lang="scss">
  .create-thread-form {
    padding: 1rem;
    border-bottom: 1px solid var(--border);
    background: transparent;
  }

  .error-banner {
    padding: 0.5rem 0.75rem;
    margin-bottom: 0.75rem;
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border-radius: 8px;
    font-size: 0.85rem;
  }

  .input-row {
    display: flex;
    gap: 0.75rem;
  }

  .avatar-wrap {
    flex-shrink: 0;
  }

  .avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    object-fit: cover;

    &.fallback {
      display: grid;
      place-items: center;
      background: var(--primary);
      color: #fff;
      font-weight: 800;
    }
  }

  .textarea-wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  textarea {
    width: 100%;
    border: none;
    background: transparent;
    color: var(--text);
    font-family: inherit;
    font-size: 1.1rem;
    resize: none;
    outline: none;
    line-height: 1.4;

    &::placeholder {
      color: var(--text-muted);
    }
  }

  .options-bar {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
    padding-top: 0.35rem;
    border-top: 1px solid rgba(0, 0, 0, 0.05);

    :global([data-theme='dark']) & {
      border-top-color: rgba(255, 255, 255, 0.05);
    }
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    border: 0;
  }

  select {
    background: var(--bg);
    color: var(--primary);
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.25rem 0.65rem;
    font-size: 0.8rem;
    font-weight: 700;
    cursor: pointer;
    outline: none;

    &:hover {
      background: var(--border);
    }
  }

  .schedule-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    background: none;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.25rem 0.65rem;
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--text-muted);
    cursor: pointer;

    &:hover,
    &.active {
      color: var(--primary);
      border-color: var(--primary);
      background: rgba(29, 155, 240, 0.08);
    }

    .svg-icon {
      width: 14px;
      height: 14px;
      fill: none;
      stroke: currentColor;
      stroke-width: 2;
    }
  }

  .schedule-inputs {
    display: flex;
    gap: 1rem;
    padding: 0.5rem;
    background: var(--border);
    border-radius: 8px;
    font-size: 0.85rem;

    label {
      display: flex;
      align-items: center;
      gap: 0.35rem;
      color: var(--text-muted);
    }

    input {
      padding: 0.2rem 0.4rem;
      border: 1px solid var(--border);
      border-radius: 4px;
      background: var(--bg);
      color: var(--text);
    }
  }

  .bottom-bar {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 0.85rem;
    margin-top: 0.5rem;
  }

  .submit-btn {
    padding: 0.45rem 1.15rem;
    border: none;
    border-radius: 999px;
    background: var(--primary);
    color: #fff;
    font-weight: 700;
    font-size: 0.9rem;
    cursor: pointer;
    transition: opacity 0.15s ease;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:hover:not(:disabled) {
      opacity: 0.9;
    }
  }
</style>
