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
    width: 100%;
    margin-top: 18px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    text-align: left;
    overflow: hidden;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 10px 14px;
    font-size: 13px;
  }
  .title {
    flex: 1;
    font-weight: 700;
  }
  .status {
    color: var(--muted);
    font-size: 12px;
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    font-size: 12px;
    padding: 0;
  }
  .link:disabled {
    opacity: 0.5;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 240px;
    overflow: auto;
    border-top: 1px solid var(--border);
  }
  li + li {
    border-top: 1px solid var(--border);
  }
  li button {
    display: flex;
    gap: 10px;
    align-items: baseline;
    width: 100%;
    background: none;
    border: none;
    color: var(--text);
    padding: 9px 14px;
    text-align: left;
    font-size: 13px;
  }
  li button:hover {
    background: rgba(255, 255, 255, 0.04);
  }
  .ref {
    font: 12px var(--mono);
    color: var(--claude-soft);
    flex-shrink: 0;
  }
  .pr-title {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .meta {
    color: var(--muted);
    font-size: 12px;
    flex-shrink: 0;
  }
</style>
