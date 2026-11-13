<script lang="ts">
  import type { BlueprintNode, Risk } from "../api";
  import { levelLabel } from "./chapters";

  interface Props {
    risks: Risk[];
    questions: string[];
    nodes: BlueprintNode[];
    /** perguntas marcadas para entrar no corpo do review */
    picked: string[];
    onnode: (id: string) => void;
    onreview: () => void;
  }

  let { risks, questions, nodes, picked = $bindable(), onnode, onreview }: Props = $props();

  const label = $derived(new Map(nodes.map((n) => [n.id, n.label])));
  const order: Record<string, number> = { high: 0, medium: 1, low: 2 };
  const sorted = $derived([...risks].sort((a, b) => order[a.level] - order[b.level]));

  function toggle(q: string): void {
    picked = picked.includes(q) ? picked.filter((p) => p !== q) : [...picked, q];
  }
</script>

<div class="cols" class:single={!risks.length || !questions.length}>
  {#if risks.length}
    <section>
      <h3>Riscos de arquitetura</h3>
      {#each sorted as r, i (i)}
        <article class={r.level}>
          <header>
            <span class="level"><i></i>{levelLabel[r.level]}</span>
            <h4>{r.title}</h4>
          </header>
          <p>{r.detail}</p>
          {#if r.node && label.has(r.node)}
            <button class="where" onclick={() => onnode(r.node)} title="Ver este componente no mapa">
              <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M6 1.5l3.9 2.25v4.5L6 10.5 2.1 8.25v-4.5z" /></svg>
              {label.get(r.node)}
            </button>
          {/if}
        </article>
      {/each}
    </section>
  {/if}

  {#if questions.length}
    <section>
      <h3>Perguntas para o autor</h3>
      <p class="sub">Marque as que quiser levar: elas entram no comentário geral do review, que você ainda edita antes de enviar.</p>
      {#each questions as q (q)}
        <label class="q" class:on={picked.includes(q)}>
          <input type="checkbox" checked={picked.includes(q)} onchange={() => toggle(q)} />
          <span>{q}</span>
        </label>
      {/each}
      <button class="btn primary go" disabled={!picked.length} onclick={onreview}>
        {picked.length ? `Levar ${picked.length} para o review` : "Marque uma pergunta"}
      </button>
    </section>
  {/if}
</div>

<style>
  .cols {
    height: 100%;
    overflow: auto;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 400px), 1fr));
    align-content: start;
    gap: 12px;
    padding: 2px 4px 8px;
  }
  section {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  h3 {
    margin: 0;
    font-size: 15px;
  }
  .sub {
    margin: -4px 0 2px;
    font-size: 12.5px;
    color: var(--muted);
    line-height: 1.45;
  }
  article {
    border: 1px solid var(--border);
    border-radius: 11px;
    padding: 11px 13px;
    background: rgba(255, 255, 255, 0.02);
  }
  article header {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  h4 {
    margin: 0;
    font-size: 14px;
    line-height: 1.35;
  }
  .level {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 2px 8px 2px 6px;
    color: var(--text);
  }
  .level i {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--muted);
  }
  .high .level i {
    background: var(--red);
  }
  .medium .level i {
    background: var(--yellow);
  }
  .low .level i {
    background: #6f9df0;
  }
  article p {
    margin: 7px 0 0;
    font-size: 13.5px;
    line-height: 1.5;
    color: #b8bdcb;
  }
  .where {
    margin-top: 9px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: 999px;
    padding: 2px 10px 2px 7px;
    font-size: 12px;
  }
  .where:hover {
    border-color: rgba(217, 119, 87, 0.55);
  }
  .where svg {
    width: 12px;
    height: 12px;
    fill: none;
    stroke: var(--muted);
    stroke-width: 1.2;
  }
  .q {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    border: 1px solid var(--border);
    border-radius: 11px;
    padding: 10px 12px;
    font-size: 13.5px;
    line-height: 1.5;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
  }
  .q:hover {
    border-color: rgba(217, 119, 87, 0.45);
  }
  .q.on {
    border-color: rgba(217, 119, 87, 0.7);
    background: rgba(217, 119, 87, 0.08);
  }
  .q input {
    margin: 3px 0 0;
    accent-color: var(--claude);
    flex-shrink: 0;
  }
  .go {
    align-self: flex-end;
    margin-top: 2px;
  }
</style>
