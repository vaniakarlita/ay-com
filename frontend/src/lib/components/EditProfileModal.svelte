<script lang="ts">
  import { userApi, uploadImage, type CurrentUser } from '../api';
  import { currentUser } from '../user';
  import { saveSession } from '../session';
  import {
    validateName,
    validateUsername,
    validateEmail,
    validateGender,
    validateDob,
    validatePassword,
    validateConfirm,
  } from '../validators';

  let {
    user,
    onClose,
    onSaved,
  }: {
    user: CurrentUser;
    onClose: () => void;
    onSaved: (updatedUser: CurrentUser) => void;
  } = $props();

  let name = $state(user.name || '');
  let username = $state(user.username || '');
  let bio = $state(user.bio || '');
  let profilePictureUrl = $state(user.profilePictureUrl || '');
  let bannerUrl = $state(user.bannerUrl || '');

  let email = $state('');
  let gender = $state('');
  let dateOfBirth = $state('');

  let showPasswordSection = $state(false);
  let oldPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');

  let uploadingAvatar = $state(false);
  let uploadingBanner = $state(false);
  let saving = $state(false);
  let errorMessage = $state('');
  let successMessage = $state('');

  let avatarInput: HTMLInputElement;
  let bannerInput: HTMLInputElement;

  async function handleAvatarChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    if (file.size > 2 * 1024 * 1024) {
      errorMessage = 'Profile picture must be 2MB or smaller';
      return;
    }
    uploadingAvatar = true;
    errorMessage = '';
    try {
      const url = await uploadImage(file, 'profile');
      profilePictureUrl = url;
    } catch (err: any) {
      errorMessage = err.message || 'Failed to upload profile picture';
    } finally {
      uploadingAvatar = false;
    }
  }

  async function handleBannerChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    if (file.size > 2 * 1024 * 1024) {
      errorMessage = 'Banner image must be 2MB or smaller';
      return;
    }
    uploadingBanner = true;
    errorMessage = '';
    try {
      const url = await uploadImage(file, 'banner');
      bannerUrl = url;
    } catch (err: any) {
      errorMessage = err.message || 'Failed to upload banner';
    } finally {
      uploadingBanner = false;
    }
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    errorMessage = '';
    successMessage = '';

    const nameErr = validateName(name);
    if (nameErr) {
      errorMessage = nameErr;
      return;
    }

    const usernameErr = validateUsername(username);
    if (usernameErr) {
      errorMessage = usernameErr;
      return;
    }

    if (email.trim()) {
      const emailErr = validateEmail(email.trim());
      if (emailErr) {
        errorMessage = emailErr;
        return;
      }
    }

    if (gender) {
      const genderErr = validateGender(gender);
      if (genderErr) {
        errorMessage = genderErr;
        return;
      }
    }

    if (dateOfBirth) {
      const dobErr = validateDob(dateOfBirth);
      if (dobErr) {
        errorMessage = dobErr;
        return;
      }
    }

    if (bio.length > 160) {
      errorMessage = 'Bio must be at most 160 characters';
      return;
    }

    if (showPasswordSection && (newPassword || oldPassword)) {
      if (!oldPassword) {
        errorMessage = 'Current password is required to set a new password';
        return;
      }
      const pwdErr = validatePassword(newPassword);
      if (pwdErr) {
        errorMessage = pwdErr;
        return;
      }
      const confirmErr = validateConfirm(newPassword, confirmPassword);
      if (confirmErr) {
        errorMessage = confirmErr;
        return;
      }
    }

    saving = true;

    try {
      const payload: any = {
        name: name.trim(),
        username: username.trim(),
        bio: bio.trim(),
        profilePictureUrl,
        bannerUrl,
      };

      if (email.trim()) payload.email = email.trim();
      if (gender) payload.gender = gender;
      if (dateOfBirth) payload.dateOfBirth = dateOfBirth;
      if (showPasswordSection && newPassword && oldPassword) {
        payload.oldPassword = oldPassword;
        payload.newPassword = newPassword;
      }

      const res = await userApi.updateProfile(payload);

      if (res.accessToken && res.refreshToken) {
        saveSession(res.accessToken, res.refreshToken);
      }

      currentUser.update((u: CurrentUser | null) => {
        if (!u) return res.user;
        return {
          ...u,
          ...res.user,
        };
      });

      onSaved(res.user);
      onClose();
    } catch (err: any) {
      errorMessage = err.message || 'Failed to update profile';
    } finally {
      saving = false;
    }
  }
</script>

<div class="modal-backdrop" onclick={onClose} role="presentation">
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
    <div class="modal-header">
      <div class="header-left">
        <button class="btn-close" type="button" onclick={onClose} aria-label="Close">
          ✕
        </button>
        <h2>Edit profile</h2>
      </div>
      <button
        class="btn-save"
        type="submit"
        form="edit-profile-form"
        disabled={saving || uploadingAvatar || uploadingBanner}
      >
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>

    {#if errorMessage}
      <div class="alert alert-error" role="alert">
        {errorMessage}
      </div>
    {/if}

    <form id="edit-profile-form" onsubmit={handleSubmit} class="form-body">
      <div class="banner-edit">
        {#if bannerUrl}
          <img src={bannerUrl} alt="Banner preview" class="banner-preview" />
        {:else}
          <div class="banner-placeholder"></div>
        {/if}
        <button
          type="button"
          class="btn-camera banner-camera"
          onclick={() => bannerInput.click()}
          disabled={uploadingBanner}
          title="Change banner"
        >
          📷
        </button>
        <input
          type="file"
          accept="image/*"
          bind:this={bannerInput}
          onchange={handleBannerChange}
          class="hidden-input"
        />
      </div>

      <div class="avatar-edit-container">
        <div class="avatar-edit">
          {#if profilePictureUrl}
            <img src={profilePictureUrl} alt="Avatar preview" class="avatar-preview" />
          {:else}
            <div class="avatar-placeholder">{name.charAt(0) || 'U'}</div>
          {/if}
          <button
            type="button"
            class="btn-camera avatar-camera"
            onclick={() => avatarInput.click()}
            disabled={uploadingAvatar}
            title="Change photo"
          >
            📷
          </button>
          <input
            type="file"
            accept="image/*"
            bind:this={avatarInput}
            onchange={handleAvatarChange}
            class="hidden-input"
          />
        </div>
      </div>

      <div class="field-group">
        <label for="edit-name">Name</label>
        <input
          id="edit-name"
          type="text"
          bind:value={name}
          placeholder="Your full name"
          required
        />
      </div>

      <div class="field-group">
        <label for="edit-username">Username</label>
        <input
          id="edit-username"
          type="text"
          bind:value={username}
          placeholder="username"
          required
        />
      </div>

      <div class="field-group">
        <div class="label-row">
          <label for="edit-bio">Bio</label>
          <span class="char-count" class:limit={bio.length > 160}>{bio.length}/160</span>
        </div>
        <textarea
          id="edit-bio"
          bind:value={bio}
          placeholder="Tell the world about yourself..."
          rows="3"
        ></textarea>
      </div>

      <div class="field-group">
        <label for="edit-email">Email (optional update)</label>
        <input
          id="edit-email"
          type="email"
          bind:value={email}
          placeholder="name@domain.com"
        />
      </div>

      <div class="row-2">
        <div class="field-group">
          <label for="edit-gender">Gender (optional)</label>
          <select id="edit-gender" bind:value={gender}>
            <option value="">Keep existing</option>
            <option value="male">Male</option>
            <option value="female">Female</option>
          </select>
        </div>

        <div class="field-group">
          <label for="edit-dob">Birth Date (optional)</label>
          <input id="edit-dob" type="date" bind:value={dateOfBirth} />
        </div>
      </div>

      <div class="password-toggle-section">
        <button
          type="button"
          class="btn-link"
          onclick={() => (showPasswordSection = !showPasswordSection)}
        >
          {showPasswordSection ? '▲ Hide Change Password' : '▼ Change Password'}
        </button>
      </div>

      {#if showPasswordSection}
        <div class="password-fields">
          <div class="field-group">
            <label for="edit-old-pass">Current Password</label>
            <input
              id="edit-old-pass"
              type="password"
              bind:value={oldPassword}
              placeholder="Enter current password"
            />
          </div>

          <div class="field-group">
            <label for="edit-new-pass">New Password</label>
            <input
              id="edit-new-pass"
              type="password"
              bind:value={newPassword}
              placeholder="At least 8 chars (upper, lower, num, symbol)"
            />
          </div>

          <div class="field-group">
            <label for="edit-confirm-pass">Confirm New Password</label>
            <input
              id="edit-confirm-pass"
              type="password"
              bind:value={confirmPassword}
              placeholder="Re-enter new password"
            />
          </div>
        </div>
      {/if}
    </form>
  </div>
</div>

<style lang="scss">
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
    background: var(--bg-surface, #ffffff);
    color: var(--text-primary, #0f1419);
    width: 100%;
    max-width: 600px;
    max-height: 90vh;
    border-radius: 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.3);
  }

  .modal-header {
    position: sticky;
    top: 0;
    background: var(--bg-surface, #ffffff);
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color, #eff3f4);

    .header-left {
      display: flex;
      align-items: center;
      gap: 16px;

      h2 {
        font-size: 1.25rem;
        font-weight: 700;
        margin: 0;
      }
    }
  }

  .btn-close {
    background: transparent;
    border: none;
    font-size: 1.1rem;
    cursor: pointer;
    color: inherit;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background: var(--hover-overlay, rgba(0, 0, 0, 0.05));
    }
  }

  .btn-save {
    background: var(--text-primary, #0f1419);
    color: var(--bg-surface, #ffffff);
    border: none;
    border-radius: 9999px;
    padding: 8px 18px;
    font-weight: 700;
    font-size: 0.95rem;
    cursor: pointer;
    transition: opacity 0.2s;

    &:hover:not(:disabled) {
      opacity: 0.9;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .alert-error {
    background: rgba(244, 33, 46, 0.1);
    color: #f4212e;
    padding: 10px 16px;
    font-size: 0.9rem;
    border-left: 4px solid #f4212e;
  }

  .form-body {
    padding-bottom: 24px;
  }

  .banner-edit {
    position: relative;
    width: 100%;
    height: 160px;
    background: var(--bg-secondary, #cfd9de);

    .banner-preview {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .banner-placeholder {
      width: 100%;
      height: 100%;
      background: var(--bg-secondary, #cfd9de);
    }
  }

  .avatar-edit-container {
    padding: 0 16px;
    margin-top: -48px;
    margin-bottom: 16px;
  }

  .avatar-edit {
    position: relative;
    width: 96px;
    height: 96px;
    border-radius: 50%;
    border: 4px solid var(--bg-surface, #ffffff);
    overflow: hidden;
    background: var(--bg-secondary, #cfd9de);

    .avatar-preview {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .avatar-placeholder {
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 2.2rem;
      font-weight: 700;
      color: var(--text-secondary, #536471);
    }
  }

  .btn-camera {
    position: absolute;
    inset: 0;
    margin: auto;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.6);
    color: #ffffff;
    border: none;
    font-size: 1.2rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s;

    &:hover:not(:disabled) {
      background: rgba(0, 0, 0, 0.8);
    }
  }

  .hidden-input {
    display: none;
  }

  .field-group {
    padding: 8px 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;

    label {
      font-size: 0.85rem;
      font-weight: 600;
      color: var(--text-secondary, #536471);
    }

    input,
    textarea,
    select {
      background: transparent;
      border: 1px solid var(--border-color, #cfd9de);
      border-radius: 8px;
      padding: 10px 12px;
      font-size: 0.95rem;
      color: inherit;
      outline: none;
      font-family: inherit;

      &:focus {
        border-color: var(--primary, #1d9bf0);
      }
    }
  }

  .label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .char-count {
    font-size: 0.8rem;
    color: var(--text-secondary, #536471);

    &.limit {
      color: #f4212e;
    }
  }

  .row-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .password-toggle-section {
    padding: 12px 16px 4px;

    .btn-link {
      background: transparent;
      border: none;
      color: var(--primary, #1d9bf0);
      font-size: 0.95rem;
      font-weight: 600;
      cursor: pointer;
      padding: 0;

      &:hover {
        text-decoration: underline;
      }
    }
  }

  .password-fields {
    margin-top: 8px;
    border-top: 1px dashed var(--border-color, #eff3f4);
    padding-top: 8px;
  }
</style>