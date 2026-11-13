<script lang="ts">
  import type { Decision } from "../api";

  interface Props {
    decisions: Decision[];
  }

  let { decisions }: Props = $props();
</script>

<div class="list">
  <div class="col">
    {#each decisions as d, i (i)}
      <article>
        <header>
          <h3>{i + 1}. {d.title}</h3>
          <span class="label">Caminho escolhido</span>
          <p class="what">{d.chosen}</p>
          {#if d.why}<p class="why">{d.why}</p>{/if}
        </header>

        <div class="scale">
          <section>
            <h4 class="label">O que ganha</h4>
            <ul>
              {#each d.gains as g (g)}<li>{g}</li>{/each}
            </ul>
          </section>
          <section>
            <h4 class="label">O que custa</h4>
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
            <h4 class="label">Caminhos que ficaram de fora</h4>
            {#each d.alternatives as a (a.name)}
              <p><b>{a.name}</b>{a.whyNot ? ": " + a.whyNot : ""}</p>
            {/each}
          </div>
        {/if}
      </article>
    {/each}
  </div>
</div>

<style>
  .list {
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
  article {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    padding-top: var(--sp-5);
    border-top: 1px solid var(--border);
  }
  header {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  h3 {
    margin: 0 0 var(--sp-1);
    font-size: var(--fs-md);
    font-weight: 600;
    line-height: 1.4;
  }
  h4 {
    margin: 0;
    font-weight: 400;
  }
  p {
    margin: 0;
  }
  .why {
    color: var(--text-2);
  }
  .scale {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--sp-5);
  }
  .scale section,
  .alts {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .none,
  .alts p {
    color: var(--muted);
  }
  .alts b {
    font-weight: 400;
    color: var(--text-2);
  }

  @media (max-width: 600px) {
    .scale {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
