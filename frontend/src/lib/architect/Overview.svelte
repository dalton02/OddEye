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
  {#if blueprint.headline}
    <section class="hero">
      <span class="kind">{prKindLabel[blueprint.kind] ?? blueprint.kind}</span>
      <h2>{blueprint.headline}</h2>
      <div class="stats">
        {#each stats as s (s.label)}
          <div class="stat"><b>{s.value}</b><span>{s.label}</span></div>
        {/each}
      </div>
    </section>
  {/if}

  {#if blueprint.before || blueprint.after}
    <section class="shift">
      <div class="side">
        <span class="tag">Antes</span>
        <p>{blueprint.before || "Não se aplica: é algo que não existia."}</p>
      </div>
      <svg class="to" viewBox="0 0 40 24" aria-hidden="true"><path d="M3 12h32M27 5l8 7-8 7" /></svg>
      <div class="side after">
        <span class="tag">Depois</span>
        <p>{blueprint.after}</p>
      </div>
    </section>
  {/if}

  {#if chapters.length > 1}
    <div class="toc">
      {#each chapters as c, i (i)}
        {#if i > 0}
          <button class="chapter" onclick={() => onjump(i)} style="animation-delay: {i * 50}ms">
            <span class="idx">{i}</span>
            <span class="main">
              <span class="title">{c.kind === "flow" ? "Fluxo: " + c.title : c.title}</span>
              <span class="teaser">{c.teaser}</span>
            </span>
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 5l7 7-7 7" /></svg>
          </button>
        {/if}
      {/each}
    </div>
  {:else}
    <p class="empty">Não há mudança de arquitetura para desenhar nesta PR. Abra a lista de arquivos se quiser ver o código.</p>
  {/if}
</div>

<style>
  .overview {
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
  }
  .hero {
    padding: 18px 22px;
  }
  .kind {
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.07em;
    text-transform: uppercase;
    color: var(--claude-soft);
  }
  h2 {
    margin: 4px 0 14px;
    font-size: 24px;
    line-height: 1.25;
    letter-spacing: -0.01em;
    text-wrap: balance;
  }
  .stats {
    display: flex;
    flex-wrap: wrap;
    gap: 10px 30px;
  }
  .stat {
    display: flex;
    flex-direction: column;
  }
  .stat b {
    font-size: 19px;
    font-variant-numeric: tabular-nums;
  }
  .stat span {
    font-size: 12px;
    color: var(--muted);
  }

  .shift {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    gap: 16px;
    padding: 16px 22px;
  }
  .tag {
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.07em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .after .tag {
    color: var(--claude-soft);
  }
  .side p {
    margin: 4px 0 0;
    font-size: 14px;
    line-height: 1.55;
    color: #b8bdcb;
  }
  .after p {
    color: var(--text);
  }
  .to {
    width: 40px;
    height: 24px;
    fill: none;
    stroke: var(--claude);
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .toc {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 320px), 1fr));
    gap: 10px;
  }
  .chapter {
    display: flex;
    align-items: center;
    gap: 13px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 12px 14px;
    color: var(--text);
    text-align: left;
    animation: rise 0.35s ease-out both;
    transition: border-color 0.15s, transform 0.15s;
  }
  .chapter:hover {
    border-color: rgba(217, 119, 87, 0.55);
    transform: translateX(3px);
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
  }
  .idx {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    background: rgba(217, 119, 87, 0.15);
    color: var(--claude-soft);
    font-weight: 800;
    font-size: 13px;
    flex-shrink: 0;
  }
  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .title {
    font-size: 14px;
    font-weight: 700;
  }
  .teaser {
    font-size: 12.5px;
    color: var(--muted);
  }
  .chapter svg {
    width: 18px;
    height: 18px;
    fill: none;
    stroke: var(--muted);
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
    flex-shrink: 0;
  }
  .empty {
    color: var(--muted);
    text-align: center;
    margin: 30px 0;
  }

  @media (max-width: 760px) {
    .shift {
      grid-template-columns: 1fr;
    }
    .to {
      transform: rotate(90deg);
      justify-self: center;
    }
  }
</style>
