<script lang="ts">
  import type { Draft, ReviewEvent } from "./api";

  interface Props {
    drafts: Draft[];
    prLabel: string;
    onsubmit: (event: ReviewEvent, body: string) => Promise<void>;
    onclose: () => void;
    onjump: (d: Draft) => void;
    /** texto que já abre no comentário geral (perguntas levadas do modo Arquiteto) */
    initialBody?: string;
  }

  let { drafts, prLabel, onsubmit, onclose, onjump, initialBody = "" }: Props = $props();

  let event = $state<ReviewEvent>("COMMENT");
  // svelte-ignore state_referenced_locally
  let body = $state(initialBody);
  let confirming = $state(false);
  let sending = $state(false);
  let error = $state("");

  const inDiff = $derived(drafts.filter((d) => d.inDiff).length);
  const outside = $derived(drafts.length - inDiff);
  const canSend = $derived(drafts.length > 0 || body.trim() !== "" || event === "APPROVE");

  const options: { value: ReviewEvent; label: string; hint: string }[] = [
    { value: "COMMENT", label: "Comentar", hint: "Feedback geral, sem aprovar nem bloquear." },
    { value: "APPROVE", label: "Aprovar", hint: "Libera a PR para merge." },
    { value: "REQUEST_CHANGES", label: "Pedir mudanças", hint: "Bloqueia o merge até os ajustes." }
  ];

  async function send(): Promise<void> {
    sending = true;
    error = "";
    try {
      await onsubmit(event, body.trim());
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      confirming = false;
    } finally {
      sending = false;
    }
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !sending) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !sending && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="review-title">
    <div class="modal-head">
      <h2 id="review-title">Enviar review</h2>
      <p>{prLabel}</p>
    </div>

    <div class="modal-body">
      <div class="block">
        <span class="label">{drafts.length} comentário{drafts.length === 1 ? "" : "s"} em rascunho</span>
        {#if drafts.length === 0}
          <p class="muted">Nenhum comentário de linha. Você ainda pode aprovar ou deixar um comentário geral.</p>
        {:else}
          <ul class="drafts">
            {#each drafts as d (d.id)}
              <li>
                <button class="jump" onclick={() => onjump(d)}>
                  <span class="loc">{d.path.split("/").pop()}:{d.startLine === d.line ? d.line : `${d.startLine}-${d.line}`}</span>
                  <span class="txt">{d.body}</span>
                  {#if !d.inDiff}<span class="out">corpo</span>{/if}
                </button>
              </li>
            {/each}
          </ul>
          {#if outside > 0}
            <p class="muted">
              {outside} comentário{outside === 1 ? "" : "s"} fora do diff entra{outside === 1 ? "" : "m"} no corpo do review, citando arquivo e linha.
            </p>
          {/if}
        {/if}
      </div>

      <label class="block">
        <span class="label">Comentário geral</span>
        <textarea class="field" bind:value={body} rows="4" placeholder="Opcional. Markdown funciona."></textarea>
      </label>

      <fieldset class="events">
        <legend class="label">Tipo de review</legend>
        {#each options as o}
          <label class:active={event === o.value}>
            <input type="radio" name="event" value={o.value} bind:group={event} />
            <span class="opt">{o.label}</span>
            <span class="hint">{o.hint}</span>
          </label>
        {/each}
      </fieldset>

      {#if error}<p class="error-text">{error}</p>{/if}
    </div>

    <div class="modal-foot">
      {#if confirming}
        <span class="confirm-text">Publicar no GitHub com a sua conta? Isso notifica o autor.</span>
        <button class="btn" disabled={sending} onclick={() => (confirming = false)}>Voltar</button>
        <button class="btn primary" disabled={sending} onclick={send}>
          {sending ? "Enviando..." : "Sim, publicar"}
        </button>
      {:else}
        <button class="btn" onclick={onclose}>Continuar revisando</button>
        <button class="btn primary" disabled={!canSend} onclick={() => (confirming = true)}>Enviar review</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .modal-body p {
    margin: 0;
  }
  .block {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .muted {
    color: var(--muted);
  }
  .drafts {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 180px;
    overflow: auto;
    border-bottom: 1px solid var(--border);
  }
  .drafts li {
    border-top: 1px solid var(--border);
  }
  .jump {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    width: 100%;
    padding: var(--sp-3) 0;
    border: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    transition: background var(--ease);
  }
  .jump:hover {
    background: var(--surface-2);
  }
  /* a lista rola e cortaria o contorno de foco desenhado para fora */
  .jump:focus-visible {
    outline-offset: -2px;
  }
  .loc {
    width: 132px;
    flex-shrink: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--mono);
    font-size: var(--fs-xs);
    color: var(--muted);
  }
  .txt {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .out {
    flex-shrink: 0;
    font-size: var(--fs-xs);
    color: var(--muted);
  }
  .events {
    min-width: 0;
    margin: 0;
    padding: 0;
    border: 0;
    border-bottom: 1px solid var(--border);
  }
  .events legend {
    padding: 0 0 var(--sp-2);
  }
  .events label {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    padding: var(--sp-3) 0;
    border-top: 1px solid var(--border);
    cursor: pointer;
  }
  .events input {
    align-self: center;
    margin: 0;
  }
  .opt {
    width: 120px;
    flex-shrink: 0;
  }
  .events label.active .opt {
    font-weight: 600;
  }
  .hint {
    color: var(--muted);
  }
  .confirm-text {
    flex: 1 1 240px;
    color: var(--text);
  }
</style>
