<script lang="ts">
  import type { Draft, ReviewEvent } from "./api";

  interface Props {
    drafts: Draft[];
    prLabel: string;
    onsubmit: (event: ReviewEvent, body: string) => Promise<void>;
    onclose: () => void;
    onjump: (d: Draft) => void;
  }

  let { drafts, prLabel, onsubmit, onclose, onjump }: Props = $props();

  let event = $state<ReviewEvent>("COMMENT");
  let body = $state("");
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
    <h2 id="review-title">Enviar review</h2>
    <p class="sub">{prLabel}</p>

    <section>
      <h3>{drafts.length} comentário{drafts.length === 1 ? "" : "s"} em rascunho</h3>
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
          <p class="muted small">
            {outside} comentário{outside === 1 ? "" : "s"} fora do diff entra{outside === 1 ? "" : "m"} no corpo do review, citando arquivo e linha.
          </p>
        {/if}
      {/if}
    </section>

    <section>
      <h3>Comentário geral</h3>
      <textarea bind:value={body} rows="4" placeholder="Opcional. Markdown funciona."></textarea>
    </section>

    <section class="events">
      {#each options as o}
        <label class:active={event === o.value} class={o.value.toLowerCase()}>
          <input type="radio" name="event" value={o.value} bind:group={event} />
          <span class="label">{o.label}</span>
          <span class="hint">{o.hint}</span>
        </label>
      {/each}
    </section>

    {#if error}<p class="error">{error}</p>{/if}

    <footer>
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
    </footer>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: rgba(2, 3, 10, 0.82);
    display: grid;
    place-items: center;
    padding: 24px;
  }
  .modal {
    width: min(640px, 100%);
    max-height: 90vh;
    overflow: auto;
    background: var(--panel-strong);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 30px 90px rgba(0, 0, 0, 0.6);
  }
  h2 {
    margin: 0;
    font-size: 20px;
  }
  .sub {
    margin: 4px 0 18px;
    color: var(--muted);
    font-size: 13px;
  }
  h3 {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  section {
    margin-bottom: 18px;
  }
  .muted {
    color: var(--muted);
    font-size: 13px;
    margin: 0;
  }
  .small {
    margin-top: 8px;
    font-size: 12px;
  }
  .drafts {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 180px;
    overflow: auto;
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .drafts li + li {
    border-top: 1px solid var(--border);
  }
  .jump {
    display: flex;
    gap: 10px;
    width: 100%;
    align-items: center;
    background: none;
    border: none;
    color: var(--text);
    padding: 8px 12px;
    text-align: left;
    font-size: 13px;
  }
  .jump:hover {
    background: rgba(255, 255, 255, 0.04);
  }
  .loc {
    font-family: var(--mono);
    color: var(--accent);
    flex-shrink: 0;
    font-size: 12px;
  }
  .txt {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .out {
    font-size: 11px;
    color: var(--yellow);
  }
  textarea {
    width: 100%;
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid var(--border);
    border-radius: 10px;
    color: var(--text);
    padding: 10px 12px;
    font: 13px/1.5 var(--ui);
    resize: vertical;
    outline: none;
  }
  textarea:focus {
    border-color: rgba(140, 160, 255, 0.45);
  }
  .events {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }
  .events label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
  }
  .events input {
    display: none;
  }
  .events .label {
    font-weight: 700;
    font-size: 14px;
  }
  .events .hint {
    font-size: 12px;
    color: var(--muted);
  }
  .events label.active.comment {
    border-color: var(--accent);
    background: rgba(138, 180, 255, 0.1);
  }
  .events label.active.approve {
    border-color: var(--green);
    background: rgba(63, 185, 80, 0.1);
  }
  .events label.active.request_changes {
    border-color: var(--red);
    background: rgba(248, 81, 73, 0.1);
  }
  .error {
    color: var(--red);
    font-size: 13px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 10px;
  }
  .confirm-text {
    flex: 1;
    font-size: 13px;
    color: var(--yellow);
  }
</style>
