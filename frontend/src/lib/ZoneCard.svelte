<script lang="ts">
  import { onMount } from "svelte";

  interface Props {
    mode: "view" | "edit";
    startLine: number;
    line: number;
    body?: string;
    inDiff: boolean;
    onsave?: (body: string) => void;
    oncancel?: () => void;
    onedit?: () => void;
    onremove?: () => void;
  }

  let { mode, startLine, line, body = "", inDiff, onsave, oncancel, onedit, onremove }: Props = $props();

  let text = $state("");
  let textarea: HTMLTextAreaElement | undefined = $state();

  const range = $derived(startLine === line ? `L${line}` : `L${startLine}–L${line}`);

  onMount(() => {
    text = body;
    if (mode === "edit") setTimeout(() => textarea?.focus(), 30);
  });

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && text.trim()) {
      e.preventDefault();
      onsave?.(text.trim());
    }
    if (e.key === "Escape") oncancel?.();
  }
</script>

<div class="card" class:edit={mode === "edit"}>
  <header>
    <span class="who">{mode === "edit" ? "Novo comentário" : "Seu rascunho"}</span>
    <span class="range">{range}</span>
    {#if !inDiff}
      <span class="warn" title="O GitHub só aceita comentário de linha em trechos do diff">
        fora do diff: vai no corpo do review
      </span>
    {/if}
    {#if mode === "view"}
      <span class="actions">
        <button class="link" onclick={() => onedit?.()}>Editar</button>
        <button class="link quiet" onclick={() => onremove?.()}>Remover</button>
      </span>
    {/if}
  </header>

  {#if mode === "edit"}
    <textarea
      class="field"
      bind:this={textarea}
      bind:value={text}
      onkeydown={onKey}
      rows="3"
      placeholder="Markdown funciona."
    ></textarea>
    <footer>
      <span class="keys">Ctrl + Enter salva · Esc cancela</span>
      <button class="btn small" onclick={() => oncancel?.()}>Cancelar</button>
      <button class="btn small primary" disabled={!text.trim()} onclick={() => onsave?.(text.trim())}>
        Salvar rascunho
      </button>
    </footer>
  {:else}
    <p class="body">{body}</p>
  {/if}
</div>

<style>
  .card {
    max-width: 720px;
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    padding: var(--sp-3) var(--sp-4);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-sm);
    background: var(--surface);
    font-family: var(--ui);
    font-size: var(--fs-sm);
    line-height: 1.5;
    color: var(--text);
  }
  .card.edit {
    border-color: var(--primary);
  }
  header {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    flex-wrap: wrap;
  }
  .who {
    font-weight: 600;
  }
  .range {
    font-family: var(--mono);
    font-size: var(--fs-xs);
    color: var(--muted);
  }
  .warn {
    font-size: var(--fs-xs);
    color: var(--yellow);
  }
  .actions {
    display: flex;
    gap: var(--sp-3);
    margin-left: auto;
  }
  textarea {
    min-height: 72px;
  }
  .body {
    margin: 0;
    white-space: pre-wrap;
  }
  footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: var(--sp-2);
  }
  .keys {
    margin-right: auto;
    font-size: var(--fs-xs);
    color: var(--muted);
  }
</style>
