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
      .map((n) => ({ label: n.label, ...n.weight }))
      .sort((a, b) => b.additions + b.deletions - (a.additions + a.deletions));
    if (loose.additions + loose.deletions > 0) rows.push({ label: "Fora dos componentes", ...loose });
    return rows;
  });
  const maxWeight = $derived(Math.max(1, ...weights.map((w) => w.additions + w.deletions)));
  const pct = (n: number): string => (n / maxWeight) * 100 + "%";
</script>

<div class="impact">
  {#if impact.length}
    <section>
      <h3>Como a PR mexe na saúde do sistema</h3>
      <p class="sub">Nota de −2 a +2 em cada eixo. Para a direita o sistema melhora, para a esquerda piora.</p>

      <div class="axes">
        <div class="row head" aria-hidden="true">
          <span></span>
          <div class="track ticks">
            <span style="left: 0">−2</span><span style="left: 25%">−1</span><span style="left: 50%">0</span><span style="left: 75%">+1</span><span
              style="left: 100%">+2</span
            >
          </div>
          <span></span>
        </div>
        {#each impact as a (a.axis)}
          <div class="row" title="{axisLabel[a.axis]}: {signed(a.score)} ({scoreLabel[String(a.score)]})">
            <span class="name">{axisLabel[a.axis]}</span>
            <div class="track">
              <i class="grid" style="left: 0"></i><i class="grid" style="left: 25%"></i><i class="grid" style="left: 75%"></i><i
                class="grid"
                style="left: 100%"
              ></i>
              <i class="zero"></i>
              {#if a.score > 0}
                <i class="bar up" style="left: 50%; width: {a.score * 25}%"></i>
              {:else if a.score < 0}
                <i class="bar down" style="right: 50%; width: {-a.score * 25}%"></i>
              {:else}
                <i class="still"></i>
              {/if}
            </div>
            <span class="note"><b>{signed(a.score)}</b> <span class="verdict">{scoreLabel[String(a.score)]}.</span> {a.note}</span>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  {#if weights.length}
    <section>
      <h3>Onde está o peso da mudança</h3>
      <p class="sub">
        Linhas alteradas em cada componente, contadas direto do diff.
        <span class="key"><i class="sw add"></i>adicionadas</span>
        <span class="key"><i class="sw del"></i>removidas</span>
      </p>
      <div class="weights">
        {#each weights as w (w.label)}
          <div class="wrow" title="{w.label}: {w.additions} linhas adicionadas, {w.deletions} removidas">
            <span class="name">{w.label}</span>
            <div class="stack">
              {#if w.additions}<i class="seg add" style="width: {pct(w.additions)}"></i>{/if}
              {#if w.deletions}<i class="seg del" style="width: {pct(w.deletions)}"></i>{/if}
              <span class="val">+{w.additions} −{w.deletions}</span>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>

<style>
  .impact {
    height: 100%;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 2px 4px 8px;
  }
  section {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 16px 20px 18px;
  }
  h3 {
    margin: 0;
    font-size: 15px;
  }
  .sub {
    margin: 3px 0 14px;
    font-size: 12.5px;
    color: var(--muted);
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
  }

  .row {
    display: grid;
    grid-template-columns: 124px minmax(160px, 300px) minmax(0, 1fr);
    align-items: center;
    column-gap: 18px;
    min-height: 34px;
  }
  .row.head {
    min-height: 20px;
  }
  .name {
    font-size: 13px;
    text-align: right;
    color: var(--text);
  }
  .track {
    position: relative;
    height: 34px;
  }
  .ticks {
    height: 18px;
  }
  .ticks span {
    position: absolute;
    top: 0;
    transform: translateX(-50%);
    font-size: 11px;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .grid {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 1px;
    background: rgba(160, 170, 210, 0.1);
  }
  .zero {
    position: absolute;
    left: 50%;
    top: 0;
    bottom: 0;
    width: 1px;
    background: rgba(160, 170, 210, 0.45);
  }
  .bar {
    position: absolute;
    top: 50%;
    height: 14px;
    margin-top: -7px;
  }
  /* par divergente frio/quente; a ponta do dado é arredondada, a base (no zero) é reta */
  .bar.up {
    background: #6f9df0;
    border-radius: 0 4px 4px 0;
  }
  .bar.down {
    background: #e5735f;
    border-radius: 4px 0 0 4px;
  }
  .still {
    position: absolute;
    left: 50%;
    top: 50%;
    width: 9px;
    height: 9px;
    margin: -4.5px 0 0 -4.5px;
    border-radius: 50%;
    background: #5d6377;
    box-shadow: 0 0 0 2px #111217;
  }
  .note {
    font-size: 12.5px;
    line-height: 1.45;
    color: #b8bdcb;
    padding: 5px 0;
  }
  .note b {
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .verdict {
    color: var(--text);
  }

  .key {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .sw {
    width: 10px;
    height: 10px;
    border-radius: 3px;
  }
  .add {
    background: var(--green);
  }
  .del {
    background: var(--red);
  }
  .weights {
    max-width: 920px;
    display: flex;
    flex-direction: column;
    gap: 9px;
  }
  .wrow {
    display: grid;
    grid-template-columns: 124px minmax(0, 1fr);
    align-items: center;
    column-gap: 18px;
  }
  .stack {
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 0;
    /* a parte que sobra à direita guarda o rótulo de valor da maior barra */
    padding-right: 96px;
  }
  .seg {
    height: 14px;
    min-width: 3px;
    flex-shrink: 0;
  }
  .seg:last-of-type {
    border-radius: 0 4px 4px 0;
  }
  .val {
    margin-left: 8px;
    margin-right: -96px;
    font-size: 12px;
    color: var(--muted);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  @media (max-width: 900px) {
    .row {
      grid-template-columns: 110px minmax(0, 1fr);
    }
    .row .note {
      grid-column: 2;
      padding-top: 0;
    }
    .row.head span:last-child {
      display: none;
    }
  }
</style>
