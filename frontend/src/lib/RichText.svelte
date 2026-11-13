<script lang="ts">
  import { parseRichText, type Resolver } from "./paths";

  interface Props {
    text: string;
    resolve: Resolver;
    onlink: (path: string) => void;
    /** sem animação de digitação (texto já lido, ou notas curtas) */
    instant?: boolean;
    ontyping?: (typing: boolean) => void;
  }

  let { text, resolve, onlink, instant = false, ontyping }: Props = $props();

  const parts = $derived(parseRichText(text, resolve));
  const total = $derived(parts.reduce((n, p) => n + p.text.length, 0));

  let shown = $state(0);
  let frame = 0;

  /* Animação por tempo, não por tick: termina em no máximo MAX_MS, independente da taxa de quadros. */
  const MS_PER_CHAR = 2.5;
  const MIN_MS = 200;
  const MAX_MS = 700;

  $effect(() => {
    const length = total;
    cancelAnimationFrame(frame);
    if (!length || instant) {
      shown = length;
      ontyping?.(false);
      return;
    }
    shown = 0;
    ontyping?.(true);
    const duration = Math.min(MAX_MS, Math.max(MIN_MS, length * MS_PER_CHAR));
    const start = performance.now();
    const step = (now: number): void => {
      const progress = Math.min(1, (now - start) / duration);
      shown = Math.ceil(length * progress);
      if (progress < 1) frame = requestAnimationFrame(step);
      else ontyping?.(false);
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  });

  export function finish(): void {
    cancelAnimationFrame(frame);
    shown = total;
    ontyping?.(false);
  }

  const visible = $derived.by(() => {
    let budget = shown;
    const out: { text: string; code: boolean; path: string | null; partial: boolean }[] = [];
    for (const p of parts) {
      if (budget <= 0) break;
      const take = Math.min(budget, p.text.length);
      out.push({ ...p, text: p.text.slice(0, take), partial: take < p.text.length });
      budget -= take;
    }
    return out;
  });
</script>

{#each visible as p, i (i)}{#if p.path && !p.partial}<button
      class="link"
      class:code={p.code}
      title="Abrir {p.path}"
      onclick={(e) => {
        e.stopPropagation();
        onlink(p.path!);
      }}>{p.text}</button
    >{:else if p.code}<code>{p.text}</code>{:else}{p.text}{/if}{/each}{#if shown < total}<span class="caret">▍</span>{/if}

<style>
  code,
  .link.code {
    font-family: var(--mono);
    font-size: 0.9em;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 4px;
    padding: 0 4px;
  }
  .link {
    display: inline;
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    color: #9cc0f5;
    text-decoration: underline;
    text-decoration-color: rgba(156, 192, 245, 0.45);
    text-underline-offset: 2px;
    cursor: pointer;
    text-align: left;
  }
  .link.code {
    background: rgba(91, 140, 214, 0.14);
  }
  .link:hover {
    color: #c3d8fa;
    text-decoration-color: currentColor;
  }
  .caret {
    color: var(--claude-soft);
    animation: caret 0.8s steps(2) infinite;
  }
  @keyframes caret {
    50% {
      opacity: 0;
    }
  }
</style>
