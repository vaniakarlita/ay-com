<script lang="ts">
  import { userApi } from '../api';

  let {
    targetUserId,
    targetUsername,
    onClose,
    onSuccess,
  }: {
    targetUserId: string;
    targetUsername: string;
    onClose: () => void;
    onSuccess?: () => void;
  } = $props();

  const reasons = [
    'Spam',
    'Harassment or bullying',
    'Hate speech',
    'Misinformation',
    'Other',
  ];

  let selectedReason = $state('Spam');
  let details = $state('');
  let submitting = $state(false);
  let errorMessage = $state('');
  let successMessage = $state('');

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (selectedReason === 'Other' && !details.trim()) {
      errorMessage = 'Please describe why you are reporting this account.';
      return;
    }

    submitting = true;
    errorMessage = '';
    try {
      await userApi.report(targetUserId, {
        reason: selectedReason,
        details: details.trim(),
      });
      successMessage = 'Thank you. Your report has been submitted for review.';
      setTimeout(() => {
        if (onSuccess) onSuccess();
        onClose();
      }, 1500);
    } catch (err: any) {
      errorMessage = err.message || 'Failed to submit report. Please try again.';
    } finally {
      submitting = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !submitting) {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class="modal-backdrop"
  onclick={(e) => {
    if (e.target === e.currentTarget && !submitting) onClose();
  }}
  role="presentation"
>
  <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="report-title">
    <div class="modal-header">
      <h2 id="report-title">Report @{targetUsername}</h2>
      <button
        type="button"
        class="close-btn"
        onclick={onClose}
        disabled={submitting}
        aria-label="Close"
      >
        ✕
      </button>
    </div>

    {#if successMessage}
      <div class="success-box">
        <p>{successMessage}</p>
      </div>
    {:else}
      <form onsubmit={handleSubmit} class="report-form">
        <p class="subtitle">
          Help us understand the issue. What is wrong with this account?
        </p>

        {#if errorMessage}
          <div class="error-box">{errorMessage}</div>
        {/if}

        <div class="reasons-list">
          {#each reasons as reason}
            <label class="reason-item">
              <input
                type="radio"
                name="report-reason"
                value={reason}
                bind:group={selectedReason}
                disabled={submitting}
              />
              <span class="reason-text">{reason}</span>
            </label>
          {/each}
        </div>

        {#if selectedReason === 'Other'}
          <div class="details-group">
            <label for="report-details" class="details-label">
              Provide additional details <span class="required">*</span>
            </label>
            <textarea
              id="report-details"
              bind:value={details}
              rows="3"
              placeholder="Explain the context or reason for this report..."
              disabled={submitting}
              class="details-input"
            ></textarea>
          </div>
        {:else}
          <div class="details-group">
            <label for="report-details" class="details-label">
              Additional details (optional)
            </label>
            <textarea
              id="report-details"
              bind:value={details}
              rows="2"
              placeholder="Optional additional notes..."
              disabled={submitting}
              class="details-input"
            ></textarea>
          </div>
        {/if}

        <div class="modal-actions">
          <button
            type="button"
            class="btn-secondary"
            onclick={onClose}
            disabled={submitting}
          >
            Cancel
          </button>
          <button type="submit" class="btn-danger" disabled={submitting}>
            {submitting ? 'Submitting...' : 'Submit Report'}
          </button>
        </div>
      </form>
    {/if}
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 16px;
  }

  .modal-card {
    background: var(--bg-card, #1e2029);
    border: 1px solid var(--border-color, #2d313e);
    border-radius: 14px;
    width: 100%;
    max-width: 480px;
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.5);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color, #2d313e);
  }

  .modal-header h2 {
    font-size: 1.15rem;
    font-weight: 700;
    margin: 0;
    color: var(--text-main, #f0f2f5);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted, #8b92a5);
    font-size: 1.1rem;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 6px;
    line-height: 1;
  }

  .close-btn:hover:not(:disabled) {
    color: var(--text-main, #f0f2f5);
    background: var(--bg-hover, #2a2d3d);
  }

  .report-form {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .subtitle {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-muted, #8b92a5);
  }

  .error-box {
    background: rgba(239, 68, 68, 0.15);
    border: 1px solid rgba(239, 68, 68, 0.4);
    color: #ef4444;
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 0.85rem;
  }

  .success-box {
    padding: 32px 20px;
    text-align: center;
    color: #22c55e;
    font-size: 1rem;
    font-weight: 600;
  }

  .reasons-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .reason-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: var(--bg-hover, #262938);
    border: 1px solid var(--border-color, #2d313e);
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }

  .reason-item:hover {
    background: var(--bg-hover-strong, #2f3447);
    border-color: var(--text-muted, #8b92a5);
  }

  .reason-item input[type='radio'] {
    accent-color: #ef4444;
    cursor: pointer;
  }

  .reason-text {
    font-size: 0.9rem;
    color: var(--text-main, #f0f2f5);
  }

  .details-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .details-label {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-main, #f0f2f5);
  }

  .required {
    color: #ef4444;
  }

  .details-input {
    width: 100%;
    background: var(--bg-hover, #262938);
    border: 1px solid var(--border-color, #2d313e);
    border-radius: 8px;
    padding: 10px;
    font-size: 0.9rem;
    color: var(--text-main, #f0f2f5);
    resize: vertical;
    font-family: inherit;
    box-sizing: border-box;
  }

  .details-input:focus {
    outline: none;
    border-color: #ef4444;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 8px;
  }

  .btn-secondary {
    padding: 8px 16px;
    border-radius: 8px;
    background: transparent;
    border: 1px solid var(--border-color, #2d313e);
    color: var(--text-main, #f0f2f5);
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--bg-hover, #2a2d3d);
  }

  .btn-danger {
    padding: 8px 18px;
    border-radius: 8px;
    background: #ef4444;
    border: none;
    color: #ffffff;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn-danger:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-danger:disabled,
  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
