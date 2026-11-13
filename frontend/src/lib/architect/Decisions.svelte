<script lang="ts">
  import type { Decision } from "../api";

  interface Props {
    decisions: Decision[];
  }

  let { decisions }: Props = $props();
</script>

<div class="list">
  {#each decisions as d, i (i)}
    <article style="animation-delay: {i * 60}ms">
      <header>
        <span class="idx">{i + 1}</span>
        <h3>{d.title}</h3>
      </header>

      <div class="chosen">
        <span class="tag">Caminho escolhido</span>
        <p class="what">{d.chosen}</p>
        {#if d.why}<p class="why">{d.why}</p>{/if}
      </div>

      <div class="scale">
        <section class="gain">
          <h4><svg viewBox="0 0 12 12" aria-hidden="true"><path d="M6 2l4 6H2z" /></svg>O que ganha</h4>
          <ul>
            {#each d.gains as g (g)}<li>{g}</li>{/each}
          </ul>
        </section>
        <section class="cost">
          <h4><svg viewBox="0 0 12 12" aria-hidden="true"><path d="M6 10L2 4h8z" /></svg>O que custa</h4>
          {#if d.costs.length}
            <ul>
              {#each d.costs as c (c)}<li>{c}</li>{/each}
            </ul>
          {:else}
            <p class="none">Nenhum custo apontado.</p>
          {/if}
        </section>
      </div>

      {#if d.alternatives.length}
        <div class="alts">
          <h4>Caminhos que ficaram de fora</h4>
          {#each d.alternatives as a (a.name)}
            <p><b>{a.name}</b>{a.whyNot ? ": " + a.whyNot : ""}</p>
          {/each}
        </div>
      {/if}
    </article>
  {/each}
</div>

<style>
  .list {
    height: 100%;
    overflow: auto;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 440px), 1fr));
    align-content: start;
    gap: 12px;
    padding: 2px 4px 8px;
  }
  article {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    animation: rise 0.35s ease-out both;
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
  }
  header {
    display: flex;
    align-items: center;
    gap: 12px;
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
  h3 {
    margin: 0;
    font-size: 15.5px;
    line-height: 1.35;
  }
  .chosen {
    border-left: 3px solid var(--claude);
    padding: 2px 0 2px 12px;
  }
  .tag {
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--claude-soft);
  }
  .what {
    margin: 3px 0 0;
    font-size: 14.5px;
    font-weight: 700;
    line-height: 1.45;
  }
  .why {
    margin: 4px 0 0;
    font-size: 13.5px;
    line-height: 1.5;
    color: #b8bdcb;
  }
  .scale {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .scale section {
    border-radius: 10px;
    padding: 10px 12px;
    border: 1px solid;
  }
  .scale .gain {
    background: rgba(63, 185, 80, 0.06);
    border-color: rgba(63, 185, 80, 0.22);
  }
  .scale .cost {
    background: rgba(236, 131, 90, 0.06);
    border-color: rgba(236, 131, 90, 0.25);
  }
  h4 {
    margin: 0 0 6px;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--muted);
  }
  h4 svg {
    width: 11px;
    height: 11px;
  }
  .gain h4 svg {
    fill: var(--green);
  }
  .cost h4 svg {
    fill: #ec835a;
  }
  ul {
    margin: 0;
    padding-left: 16px;
    font-size: 13px;
    line-height: 1.5;
  }
  li + li {
    margin-top: 3px;
  }
  .none {
    margin: 0;
    font-size: 13px;
    color: var(--muted);
  }
  .alts {
    border-top: 1px dashed var(--border);
    padding-top: 10px;
  }
  .alts p {
    margin: 0 0 4px;
    font-size: 13px;
    line-height: 1.5;
    color: #b8bdcb;
  }
  .alts b {
    color: var(--text);
  }
</style>
