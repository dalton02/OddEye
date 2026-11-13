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
  /* modificadores do .sev global: alto ganha a bolinha amarela, médio a cinza, baixo fica só no texto */
  const sev: Record<string, string> = { high: "high", medium: "medium", low: "none" };

  function toggle(q: string): void {
    picked = picked.includes(q) ? picked.filter((p) => p !== q) : [...picked, q];
  }
</script>

<div class="risks">
  <div class="cols">
    {#if risks.length}
      <section>
        <h3>Riscos de arquitetura</h3>
        <div class="rows">
          {#each sorted as r, i (i)}
            <article>
              <header>
                <span class="sev {sev[r.level] ?? 'medium'}">{levelLabel[r.level]}</span>
                <h4>{r.title}</h4>
              </header>
              <p>{r.detail}</p>
              {#if r.node && label.has(r.node)}
                <button class="link where" onclick={() => onnode(r.node)} title="Ver este componente no mapa">Ver {label.get(r.node)} no mapa</button>
              {/if}
            </article>
          {/each}
        </div>
      </section>
    {/if}

    {#if questions.length}
      <section>
        <h3>Perguntas para o autor</h3>
        <p class="sub">Marque as que quiser levar: elas entram no comentário geral do review, que você ainda edita antes de enviar.</p>
        <div class="rows">
          {#each questions as q (q)}
            <label class="q">
              <input type="checkbox" checked={picked.includes(q)} onchange={() => toggle(q)} />
              <span>{q}</span>
            </label>
          {/each}
        </div>
        <button class="btn go" disabled={!picked.length} onclick={onreview}>
          {picked.length ? `Levar ${picked.length} para o review` : "Marque uma pergunta"}
        </button>
      </section>
    {/if}
  </div>
</div>

<style>
  .risks {
    height: 100%;
    overflow: auto;
  }
  .cols {
    width: 100%;
    max-width: 1040px;
    margin: 0 auto;
    padding-bottom: var(--sp-6);
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-6) 48px;
  }
  section {
    flex: 1 1 360px;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  h3 {
    margin: 0 0 var(--sp-3);
    font-size: var(--fs-md);
    font-weight: 600;
  }
  h3:has(+ .sub) {
    margin-bottom: var(--sp-1);
  }
  .sub {
    margin: 0 0 var(--sp-3);
    color: var(--muted);
  }
  .rows {
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--border);
  }
  article {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
    padding: var(--sp-4) 0;
    border-top: 1px solid var(--border);
  }
  article header {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
  }
  h4 {
    margin: 0;
    font-size: var(--fs-sm);
    font-weight: 600;
  }
  article p {
    margin: 0;
    color: var(--text-2);
  }
  .where {
    align-self: flex-start;
  }
  .q {
    display: flex;
    align-items: flex-start;
    gap: var(--sp-3);
    padding: var(--sp-3) 0;
    border-top: 1px solid var(--border);
    cursor: pointer;
  }
  .q input {
    width: 16px;
    height: 16px;
    margin: 2px 0 0;
    flex-shrink: 0;
  }
  .go {
    align-self: flex-start;
    margin-top: var(--sp-4);
  }
</style>
