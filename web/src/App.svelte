<script lang="ts">
  import V1App from './v1/V1App.svelte';
  import V2App from './v2/V2App.svelte';
  import type { Lang, Theme } from './data';

  // Two front-of-house demos share fixture data + theme/language state:
  //   v2 (default) — memento-first: a detailed memento "page" + preview carousel.
  //   v1 (#map)    — the liuaaron-aligned map reader, reached as the "more" view.
  type Version = 'v1' | 'v2';

  function fromHash(): Version {
    return location.hash === '#map' || location.hash === '#v1' ? 'v1' : 'v2';
  }

  let version: Version = fromHash();
  let lang: Lang = 'ja';
  let theme: Theme = 'dark';

  function go(next: Version) {
    version = next;
    location.hash = next === 'v1' ? '#map' : '';
  }
</script>

<svelte:window on:hashchange={() => (version = fromHash())} />

{#if version === 'v2'}
  <V2App bind:lang bind:theme toMap={() => go('v1')} />
{:else}
  <V1App bind:lang bind:theme toMemories={() => go('v2')} />
{/if}
