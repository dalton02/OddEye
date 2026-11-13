<script lang="ts">
  import type { Blueprint, Session } from "../api";
  import { prKindLabel, type Chapter } from "./chapters";

  interface Props {
    blueprint: Blueprint;
    session: Session;
    chapters: Chapter[];
    onjump: (index: number) => void;
  }

  let { blueprint, session, chapters, onjump }: Props = $props();

  const count = (change: string): number => blueprint.nodes.filter((n) => n.change === change).length;

  const stats = $derived(
    [
      { value: String(session.totalFiles), label: session.totalFiles === 1 ? "arquivo na PR" : "arquivos na PR" },
      { value: `+${session.additions} −${session.deletions}`, label: "linhas" },
      { value: String(count("new")), label: count("new") === 1 ? "componente novo" : "componentes novos" },
      { value: String(count("changed")), label: count("changed") === 1 ? "componente alterado" : "componentes alterados" },
      { value: String(count("removed")), label: count("removed") === 1 ? "componente removido" : "componentes removidos" }
    ].filter((s, i) => i < 2 || s.value !== "0")
  );
</script>

<div class="overview">
  <div class="col">
    {#if blueprint.headline}
      <section class="hero">
        <p class="label meta">
          {prKindLabel[blueprint.kind] ?? blueprint.kind}{#each stats as s, i (s.label)}{" · "}{#if i === 1}<span class="add">+{session.additions}</span
              >{" "}<span class="del">−{session.deletions}</span>{:else}{s.value}{/if}{" "}{s.label}{/each}
        </p>
        <h2>{blueprint.headline}</h2>
      </section>
    {/if}

    {#if blueprint.before || blueprint.after}
      <section class="shift">
        <div class="side">
          <span class="label">Antes</span>
          <p>{blueprint.before || "Não se aplica: é algo que não existia."}</p>
        </div>
        <div class="side after">
          <span class="label">Depois</span>
          <p>{blueprint.after}</p>
        </div>
      </section>
    {/if}

    {#if chapters.length > 1}
      <div class="toc">
        {#each chapters as c, i (i)}
          {#if i > 0}
            <button class="chapter" onclick={() => onjump(i)}>
              <span class="idx">{i}</span>
              <span class="title">{c.kind === "flow" ? "Fluxo: " + c.title : c.title}</span>
              <span class="teaser">{c.teaser}</span>
            </button>
          {/if}
        {/each}
      </div>
    {:else}
      <p class="empty">Não há mudança de arquitetura para desenhar nesta PR. Abra a lista de arquivos se quiser ver o código.</p>
    {/if}
  </div>
</div>

<style>
  .overview {
    height: 100%;
    overflow: auto;
  }
  .col {
    width: 100%;
    max-width: 880px;
    margin: 0 auto;
    padding-bottom: var(--sp-6);
    display: flex;
    flex-direction: column;
    gap: var(--sp-6);
  }
  .hero {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .meta {
    margin: 0;
    font-variant-numeric: tabular-nums;
  }
  h2 {
    margin: 0;
    font-size: var(--fs-xl);
    font-weight: 600;
    line-height: 1.2;
    text-wrap: balance;
  }

  .shift {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--sp-5);
  }
  .side {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
    padding-top: var(--sp-3);
    border-top: 1px solid var(--border);
  }
  .side.after {
    border-top-color: var(--text);
  }
  .side p {
    margin: 0;
  }

  .toc {
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--border);
  }
  .chapter {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
    padding: var(--sp-3) 0;
    border: none;
    border-top: 1px solid var(--border);
    background: none;
    color: var(--text);
    font-size: var(--fs-sm);
    text-align: left;
  }
  .idx {
    width: 16px;
    flex-shrink: 0;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .title {
    width: 220px;
    flex-shrink: 0;
    font-weight: 600;
  }
  .teaser {
    flex: 1;
    min-width: 0;
    color: var(--muted);
    transition: color var(--ease);
  }
  .chapter:hover .teaser {
    color: var(--text);
  }
  .empty {
    margin: 0;
    padding-top: var(--sp-4);
    border-top: 1px solid var(--border);
    color: var(--muted);
  }

  @media (max-width: 760px) {
    .shift {
      grid-template-columns: minmax(0, 1fr);
    }
    .chapter {
      flex-wrap: wrap;
      row-gap: 2px;
    }
    .title {
      width: auto;
      flex: 1;
    }
    .teaser {
      flex-basis: 100%;
      padding-left: var(--sp-6);
    }
  }
</style>
