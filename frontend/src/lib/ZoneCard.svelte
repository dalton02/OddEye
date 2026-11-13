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
    <span class="who">
      <span class="dot"></span>
      {mode === "edit" ? "Novo comentário" : "Seu rascunho"}
    </span>
    <span class="range">{range}</span>
    {#if !inDiff}
      <span class="warn" title="O GitHub só aceita comentário de linha em trechos do diff">
        fora do diff: vai no corpo do review
      </span>
    {/if}
  </header>

  {#if mode === "edit"}
    <textarea
      bind:this={textarea}
      bind:value={text}
      onkeydown={onKey}
      rows="3"
      placeholder="Escreva seu comentário (Markdown funciona). Ctrl+Enter salva, Esc cancela."
    ></textarea>
    <footer>
      <button class="btn" onclick={() => oncancel?.()}>Cancelar</button>
      <button class="btn primary" disabled={!text.trim()} onclick={() => onsave?.(text.trim())}>
        Salvar rascunho
      </button>
    </footer>
  {:else}
    <p class="body">{body}</p>
    <footer>
      <button class="link" onclick={() => onedit?.()}>Editar</button>
      <button class="link danger" onclick={() => onremove?.()}>Remover</button>
    </footer>
  {/if}
</div>

<style>
  .card {
    max-width: 760px;
    background: #1d1e23;
    border: 1px solid #2c2e35;
    border-left: 3px solid var(--claude);
    border-radius: 8px;
    padding: 10px 12px;
    font-family: var(--ui);
    color: var(--text);
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.35);
  }
  .card.edit {
    border-left-color: #2f81f7;
  }
  header {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 12px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }
  .who {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 700;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--claude);
  }
  .edit .dot {
    background: #2f81f7;
  }
  .range {
    font-family: var(--mono);
    color: var(--muted);
  }
  .warn {
    color: var(--yellow);
    background: rgba(227, 179, 65, 0.12);
    border-radius: 6px;
    padding: 1px 6px;
  }
  textarea {
    width: 100%;
    resize: vertical;
    min-height: 70px;
    background: #16171b;
    color: var(--text);
    border: 1px solid #2c2e35;
    border-radius: 6px;
    padding: 8px 10px;
    font: 13px/1.5 var(--ui);
    outline: none;
  }
  textarea:focus {
    border-color: #2f81f7;
  }
  .body {
    margin: 0;
    white-space: pre-wrap;
    font-size: 13px;
    line-height: 1.5;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 8px;
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    font-size: 12px;
    padding: 2px 4px;
  }
  .link.danger {
    color: var(--red);
  }
</style>
