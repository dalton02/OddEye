<script lang="ts">
  import { api, modeLabel, prRefLabel, type Mode, type OpenPR, type ProjectPRs } from "./api";

  interface Props {
    projects: ProjectPRs[] | null;
    loading: boolean;
    error: string;
    onanalyze: (pr: OpenPR, mode: Mode) => void;
    onrefresh: () => void;
    onsettings: () => void;
  }

  let { projects, loading, error, onanalyze, onrefresh, onsettings }: Props = $props();

  const modes: Mode[] = ["code", "architect"];

  function ago(iso: string): string {
    const t = Date.parse(iso);
    if (!t || t < 0) return "";
    const min = Math.round((Date.now() - t) / 60000);
    if (min < 1) return "agora";
    if (min < 60) return `há ${min} min`;
    const h = Math.round(min / 60);
    return h < 24 ? `há ${h} h` : `há ${Math.round(h / 24)} d`;
  }

  const total = $derived(projects?.reduce((n, p) => n + p.prs.length, 0) ?? 0);
</script>

<div class="projects">
  <div class="head">
    <span class="title">
      {#if projects && projects.length}
        {total} PR{total === 1 ? "" : "s"} aberta{total === 1 ? "" : "s"}
      {:else}
        PRs abertas por projeto
      {/if}
    </span>
    <span class="status" title={error}>{loading ? "Carregando..." : error ? "Falha ao buscar as PRs" : ""}</span>
    {#if projects && projects.length}
      <button class="link" onclick={onrefresh} disabled={loading}>Atualizar</button>
    {/if}
    <button class="link" onclick={onsettings}>{projects && projects.length ? "Configurar" : "Escolher repositórios"}</button>
  </div>

  {#if projects && projects.length === 0 && !loading}
    <p class="empty">Escolha os repositórios nas Configurações: cada um vira um projeto aqui, com as PRs abertas.</p>
  {/if}

  {#each projects ?? [] as p (p.repo)}
    <section>
      <h3><span class="mono">{p.repo}</span><span class="count">{p.prs.length}</span></h3>
      {#if p.prs.length === 0}
        <p class="empty">Nenhuma PR aberta.</p>
      {:else}
        <ul>
          {#each p.prs as pr (prRefLabel(pr.ref))}
            <li>
              <button class="open" onclick={() => api.openURL(pr.url)} title="Abrir no GitHub">
                <span class="ref">#{pr.ref.number}</span>
                <span class="pr-title">{pr.title}</span>
                {#if pr.isDraft}<span class="tag">rascunho</span>{/if}
                <span class="meta">{pr.author} · {ago(pr.createdAt)}</span>
              </button>
              <span class="actions">
                {#each modes as m (m)}
                  <button class="btn" onclick={() => onanalyze(pr, m)} title="Analisar no modo {modeLabel[m]}">{modeLabel[m]}</button>
                {/each}
              </span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/each}
</div>

<style>
  .projects {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    width: 100%;
    text-align: left;
  }
  .head {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-3);
    padding-bottom: var(--sp-3);
    border-bottom: 1px solid var(--border);
  }
  .title {
    font-size: var(--fs-md);
    font-weight: 600;
  }
  .status {
    flex-grow: 1;
    color: var(--muted);
  }
  h3 {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    margin: 0 0 var(--sp-2);
    font-size: var(--fs-md);
    font-weight: 600;
  }
  .count {
    color: var(--muted);
    font-weight: 400;
  }
  .empty {
    margin: 0;
    color: var(--muted);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 240px;
    overflow: auto;
  }
  li {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    border-bottom: 1px solid var(--border);
  }
  .open {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
    flex-grow: 1;
    min-width: 0;
    padding: var(--sp-3) 0;
    border: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    transition: background var(--ease);
  }
  .open:hover {
    background: var(--surface);
  }
  /* a lista rola e cortaria o contorno de foco desenhado para fora */
  li button:focus-visible {
    outline-offset: -2px;
  }
  .ref {
    flex-shrink: 0;
    font-family: var(--mono);
    color: var(--muted);
  }
  .pr-title {
    flex-grow: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .tag {
    flex-shrink: 0;
    padding: 0 var(--sp-2);
    border: 1px solid var(--border);
    color: var(--muted);
    font-size: var(--fs-sm);
  }
  .meta {
    flex-shrink: 0;
    color: var(--muted);
    white-space: nowrap;
  }
  .actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--sp-2);
  }
</style>
