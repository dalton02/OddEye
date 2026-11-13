<script lang="ts">
  import { prRefLabel, type OpenPR, type WatcherSnapshot } from "./api";

  interface Props {
    snapshot: WatcherSnapshot | null;
    onopen: (pr: OpenPR) => void;
    oncheck: () => void;
    onsettings: () => void;
  }

  let { snapshot, onopen, oncheck, onsettings }: Props = $props();

  function ago(iso: string): string {
    const t = Date.parse(iso);
    if (!t || t < 0) return "";
    const min = Math.round((Date.now() - t) / 60000);
    if (min < 1) return "agora";
    if (min < 60) return `há ${min} min`;
    const h = Math.round(min / 60);
    return h < 24 ? `há ${h} h` : `há ${Math.round(h / 24)} d`;
  }

  const status = $derived.by(() => {
    if (!snapshot) return "";
    if (snapshot.checking) return "Verificando...";
    if (snapshot.lastError) return "Falha na última verificação";
    const last = ago(snapshot.lastPoll);
    return last ? `Verificado ${last}` : "";
  });
</script>

<div class="inbox">
  <div class="head">
    <span class="title">
      {#if snapshot && snapshot.pending.length}
        {snapshot.pending.length} PR{snapshot.pending.length === 1 ? "" : "s"} nova{snapshot.pending.length === 1 ? "" : "s"}
      {:else if snapshot && snapshot.repos > 0}
        Nenhuma PR nova em {snapshot.repos} repositório{snapshot.repos === 1 ? "" : "s"}
      {:else}
        Vigie seus repositórios e receba aviso de PR nova
      {/if}
    </span>
    <span class="status" title={snapshot?.lastError ?? ""}>{status}</span>
    {#if snapshot && snapshot.repos > 0}
      <button class="link" onclick={oncheck} disabled={snapshot.checking}>Verificar agora</button>
    {/if}
    <button class="link" onclick={onsettings}>{snapshot && snapshot.repos > 0 ? "Configurar" : "Escolher repositórios"}</button>
  </div>
  {#if snapshot && snapshot.pending.length}
    <ul>
      {#each snapshot.pending as pr (prRefLabel(pr.ref))}
        <li>
          <button onclick={() => onopen(pr)}>
            <span class="ref">{pr.ref.repo}#{pr.ref.number}</span>
            <span class="pr-title">{pr.title}</span>
            <span class="meta">{pr.author} · {ago(pr.createdAt)}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .inbox {
    display: flex;
    flex-direction: column;
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
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 240px;
    overflow: auto;
  }
  li {
    border-bottom: 1px solid var(--border);
  }
  li button {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
    width: 100%;
    padding: var(--sp-3) 0;
    border: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    transition: background var(--ease);
  }
  li button:hover {
    background: var(--surface);
  }
  /* a lista rola e cortaria o contorno de foco desenhado para fora */
  li button:focus-visible {
    outline-offset: -2px;
  }
  .ref {
    width: 132px;
    flex-shrink: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
  .meta {
    flex-shrink: 0;
    color: var(--muted);
    white-space: nowrap;
  }
</style>
