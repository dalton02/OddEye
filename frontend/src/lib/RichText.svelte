<script lang="ts">
  import { parseRichText, type Resolver } from "./paths";

  interface Props {
    text: string;
    resolve: Resolver;
    onlink: (path: string) => void;
  }

  let { text, resolve, onlink }: Props = $props();

  const parts = $derived(parseRichText(text, resolve));
</script>

{#each parts as p, i (i)}{#if p.path}<button
      class="link"
      class:code={p.code}
      title="Abrir {p.path}"
      onclick={(e) => {
        e.stopPropagation();
        onlink(p.path!);
      }}>{p.text}</button
    >{:else if p.code}<code>{p.text}</code>{:else}{p.text}{/if}{/each}

<style>
  code,
  .link.code {
    font-family: var(--mono);
    font-size: 0.88em;
  }
  code {
    color: var(--primary);
  }
</style>
