<script lang="ts">
  import type { BlueprintNode, ImpactAxis } from "../api";
  import { axisLabel } from "./chapters";

  interface Props {
    impact: ImpactAxis[];
    nodes: BlueprintNode[];
    /** linhas alteradas em arquivos que não pertencem a nenhum componente do mapa */
    loose: { additions: number; deletions: number };
  }

  let { impact, nodes, loose }: Props = $props();

  const scoreLabel: Record<string, string> = {
    "-2": "piora muito",
    "-1": "piora um pouco",
    "0": "não muda",
    "1": "melhora um pouco",
    "2": "melhora muito"
  };
  const signed = (n: number): string => (n > 0 ? "+" + n : n < 0 ? "−" + -n : "0");

  const weights = $derived.by(() => {
    const rows = nodes
      .filter((n) => n.weight.additions + n.weight.deletions > 0)
      .map((n) => ({ label: n.label, loose: false, ...n.weight }))
      .sort((a, b) => b.additions + b.deletions - (a.additions + a.deletions));
    if (loose.additions + loose.deletions > 0) rows.push({ label: "Fora dos componentes", loose: true, ...loose });
    return rows;
  });
  const maxWeight = $derived(Math.max(1, ...weights.map((w) => w.additions + w.deletions)));
  const pct = (n: number): string => (n / maxWeight) * 100 + "%";
</script>

<div class="impact">
  <div class="col">
    {#if impact.length}
      <section>
        <div class="head">
          <h3>Como a PR mexe na saúde do sistema</h3>
          <p class="label">Nota de −2 a +2 em cada eixo. Para a direita o sistema melhora, para a esquerda piora.</p>
        </div>

        {#each impact as a (a.axis)}
          <div class="row" title="{axisLabel[a.axis]}: {signed(a.score)} ({scoreLabel[String(a.score)]})">
            <span class="name">{axisLabel[a.axis]}</span>
            <div class="track" aria-hidden="true">
              {#if a.score > 0}
                <i class="bar up" style="width: {a.score * 25}%"></i>
              {:else if a.score < 0}
                <i class="bar down" style="width: {-a.score * 25}%"></i>
              {/if}
              <i class="zero"></i>
            </div>
            <b class="score">{signed(a.score)}</b>
            <span class="note">{scoreLabel[String(a.score)]}. {a.note}</span>
          </div>
        {/each}
      </section>
    {/if}

    {#if weights.length}
      <section>
        <div class="head">
          <h3>Onde está o peso da mudança</h3>
          <p class="label">
            Linhas alteradas em cada componente, contadas direto do diff.
            <span class="key"><i class="sw pos"></i>adicionadas</span>
            <span class="key"><i class="sw neg"></i>removidas</span>
          </p>
        </div>

        {#each weights as w (w.label)}
          <div class="wrow" title="{w.label}: {w.additions} linhas adicionadas, {w.deletions} removidas">
            <span class="name" class:quiet={w.loose}>{w.label}</span>
            <div class="stack" aria-hidden="true">
              {#if w.additions}<i class="part pos" style="width: {pct(w.additions)}"></i>{/if}
              {#if w.deletions}<i class="part neg" style="width: {pct(w.deletions)}"></i>{/if}
            </div>
            <span class="val mono"><span class="add">+{w.additions}</span> <span class="del">−{w.deletions}</span></span>
          </div>
        {/each}
      </section>
    {/if}
  </div>
</div>

<style>
  .impact {
    height: 100%;
    overflow: auto;
  }
  .col {
    width: 100%;
    max-width: 1040px;
    margin: 0 auto;
    padding-bottom: var(--sp-6);
    display: flex;
    flex-direction: column;
    gap: var(--sp-6);
  }
  section {
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    padding-top: var(--sp-5);
    border-top: 1px solid var(--border);
  }
  .head {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-1) var(--sp-4);
  }
  h3 {
    flex: 1;
    margin: 0;
    font-size: var(--fs-md);
    font-weight: 600;
    white-space: nowrap;
  }
  .head p {
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-1) var(--sp-3);
  }

  .row {
    display: grid;
    grid-template-columns: 140px 240px 24px minmax(0, 1fr);
    align-items: center;
    column-gap: var(--sp-4);
  }
  .track {
    position: relative;
    height: 12px;
    background: var(--surface-2);
  }
  .bar {
    position: absolute;
    top: 0;
    bottom: 0;
  }
  .bar.up {
    left: 50%;
    background: var(--text);
  }
  .bar.down {
    right: 50%;
    background: var(--red);
  }
  .zero {
    position: absolute;
    left: 50%;
    top: -3px;
    bottom: -3px;
    width: 1px;
    background: var(--muted);
  }
  .score {
    font-weight: 600;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .note {
    min-width: 0;
    color: var(--muted);
  }

  .key {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .sw {
    width: 8px;
    height: 8px;
  }
  .pos {
    background: var(--green);
  }
  .neg {
    background: var(--red);
  }
  .wrow {
    display: grid;
    grid-template-columns: 180px minmax(0, 300px) auto;
    align-items: center;
    justify-content: start;
    column-gap: var(--sp-4);
  }
  .name.quiet {
    color: var(--muted);
  }
  .stack {
    display: flex;
    gap: 2px;
    min-width: 0;
    height: 12px;
    padding-right: 2px;
  }
  .part {
    min-width: 2px;
  }
  .val {
    font-size: var(--fs-xs);
    white-space: nowrap;
  }

  @media (max-width: 900px) {
    .row {
      grid-template-columns: 140px minmax(0, 240px) 24px;
      row-gap: 2px;
    }
    .row .note {
      grid-column: 1 / -1;
      padding-bottom: var(--sp-1);
    }
    .wrow {
      grid-template-columns: 140px minmax(0, 1fr) auto;
    }
  }
</style>
