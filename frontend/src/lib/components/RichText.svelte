<script lang="ts">
  import { navigate } from '../router';

  let { text = '' }: { text: string } = $props();

  type Token = {
    type: 'text' | 'mention' | 'hashtag';
    value: string;
    payload?: string;
  };

  const tokens = $derived.by<Token[]>(() => {
    if (!text) return [];

    // Regex to capture mentions (@username) and hashtags (#hashtag)
    const regex = /(@[A-Za-z0-9_]+)|(#[A-Za-z0-9_]+)/g;
    const parts: Token[] = [];
    let lastIndex = 0;
    let match: RegExpExecArray | null;

    while ((match = regex.exec(text)) !== null) {
      if (match.index > lastIndex) {
        parts.push({
          type: 'text',
          value: text.slice(lastIndex, match.index),
        });
      }

      if (match[1]) {
        // Mention: strip '@'
        const username = match[1].slice(1);
        parts.push({
          type: 'mention',
          value: match[1],
          payload: username,
        });
      } else if (match[2]) {
        // Hashtag: strip '#'
        const tag = match[2].slice(1);
        parts.push({
          type: 'hashtag',
          value: match[2],
          payload: tag,
        });
      }

      lastIndex = regex.lastIndex;
    }

    if (lastIndex < text.length) {
      parts.push({
        type: 'text',
        value: text.slice(lastIndex),
      });
    }

    return parts;
  });

  function onMentionClick(e: MouseEvent, username: string) {
    e.stopPropagation();
    e.preventDefault();
    navigate(`/profile?u=${encodeURIComponent(username)}`);
  }

  function onHashtagClick(e: MouseEvent, tag: string) {
    e.stopPropagation();
    e.preventDefault();
    navigate(`/explore?q=${encodeURIComponent('#' + tag)}`);
  }
</script>

<span class="rich-text">
  {#each tokens as token}
    {#if token.type === 'mention'}
      <button
        type="button"
        class="token-mention"
        onclick={(e) => onMentionClick(e, token.payload ?? '')}
      >
        {token.value}
      </button>
    {:else if token.type === 'hashtag'}
      <button
        type="button"
        class="token-hashtag"
        onclick={(e) => onHashtagClick(e, token.payload ?? '')}
      >
        {token.value}
      </button>
    {:else}
      {token.value}
    {/if}
  {/each}
</span>

<style lang="scss">
  .rich-text {
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.5;
  }

  button.token-mention,
  button.token-hashtag {
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    font: inherit;
    cursor: pointer;
    text-decoration: none;
    font-weight: 600;

    &:hover {
      text-decoration: underline;
    }
  }

  button.token-mention {
    color: var(--primary);
  }

  button.token-hashtag {
    color: #0284c7; // Bright blue hashtag

    :global([data-theme='dark']) & {
      color: #38bdf8;
    }
  }
</style>
