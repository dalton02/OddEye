<script lang="ts">
  import { severityLabel, type Draft, type Session } from "./api";

  interface Props {
    session: Session;
    activePath: string | null;
    visited: Set<string>;
    drafts: Draft[];
    onopen: (path: string) => void;
    onclose: () => void;
  }

  let { session, activePath, visited, drafts, onopen, onclose }: Props = $props();

  let showSkipped = $state(false);

  function countFor(path: string): number {
    return drafts.filter((d) => d.path === path).length;
  }

  function name(path: string): string {
    return path.split("/").pop() ?? path;
  }

  function dir(path: string): string {
    const parts = path.split("/");
    parts.pop();
    return parts.join("/");
  }
</script>

<aside>
  <header>
    <h2>Arquivos da PR</h2>
    <button class="close" onclick={onclose} aria-label="Fechar">
      <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M3 3l8 8M11 3l-8 8" /></svg>
    </button>
  </header>

  <div class="scroll">
    {#if session.mode === "code"}
      <h3>Selecionados pelo Claude <span>{session.files.length}</span></h3>
    {/if}
    {#each session.files as f (f.path)}
      <button class="item" class:active={activePath === f.path} class:seen={visited.has(f.path)} onclick={() => onopen(f.path)}>
        <span class="sev {f.severity}" title={severityLabel[f.severity]}></span>
        <span class="info">
          <span class="name">{name(f.path)}</span>
          <span class="dir">{dir(f.path)}</span>
        </span>
        {#if countFor(f.path)}<span class="cm">{countFor(f.path)} rascunho{countFor(f.path) === 1 ? "" : "s"}</span>{/if}
      </button>
    {/each}

    {#if session.others.length}
      <h3>{session.mode === "code" ? "Outros alterados" : "Arquivos alterados"} <span>{session.others.length}</span></h3>
      {#each session.others as f (f.path)}
        <button class="item plain" class:active={activePath === f.path} onclick={() => onopen(f.path)}>
          <span class="info">
            <span class="name">{name(f.path)}</span>
            <span class="dir">{dir(f.path)}</span>
          </span>
          {#if countFor(f.path)}<span class="cm">{countFor(f.path)} rascunho{countFor(f.path) === 1 ? "" : "s"}</span>{/if}
          <span class="diff"><b class="add">+{f.additions}</b> <i class="del">−{f.deletions}</i></span>
        </button>
      {/each}
    {/if}

    {#if session.skipped.length}
      <button class="toggle" aria-expanded={showSkipped} onclick={() => (showSkipped = !showSkipped)}>
        <svg class:open={showSkipped} viewBox="0 0 12 12" aria-hidden="true"><path d="M4.5 2.5l3.5 3.5-3.5 3.5" /></svg>
        <span class="grow">Descartados (testes, docs, gerados)</span>
        <span>{session.skipped.length}</span>
      </button>
      {#if showSkipped}
        {#each session.skipped as s (s.path)}
          <button class="item plain dim" class:active={activePath === s.path} onclick={() => onopen(s.path)}>
            <span class="info">
              <span class="name">{name(s.path)}</span>
              <span class="dir">{dir(s.path)}</span>
            </span>
            <span class="reason">{s.reason}</span>
          </button>
        {/each}
      {/if}
    {/if}
  </div>
</aside>

<style>
  aside {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 380px;
    max-width: 100%;
    z-index: var(--z-drawer);
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--border-strong);
    background: var(--surface);
  }
  header {
    display: flex;
    align-items: center;
    height: 52px;
    padding: 0 var(--sp-2) 0 20px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  h2 {
    flex: 1;
    margin: 0;
    font-size: var(--fs-md);
    font-weight: 600;
  }
  .close {
    width: 36px;
    height: 36px;
    display: grid;
    place-items: center;
    border: none;
    border-radius: var(--r-sm);
    background: none;
    color: var(--text);
    transition: background var(--ease);
  }
  .close:hover {
    background: var(--selected);
  }
  svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .scroll {
    overflow: auto;
    padding: var(--sp-2) 0 var(--sp-5);
  }
  h3 {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin: 0;
    padding: var(--sp-4) 20px 6px;
    font-size: var(--fs-xs);
    font-weight: 400;
    color: var(--muted);
  }
  .item {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    width: 100%;
    padding: var(--sp-2) 20px;
    border: none;
    background: none;
    color: var(--text);
    font-size: var(--fs-sm);
    text-align: left;
    transition: background var(--ease);
  }
  .item.plain {
    padding-left: 40px;
  }
  .item:hover {
    background: var(--surface-2);
  }
  .item.active {
    background: var(--selected);
  }
  .item.active .name {
    font-weight: 600;
  }
  /* já visitado: o nome recua para cinza, e o que falta ver continua em destaque */
  .item.seen:not(.active) .name {
    color: var(--muted);
  }
  .item.dim {
    color: var(--muted);
  }
  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .dir {
    font-family: var(--mono);
    font-size: var(--fs-xs);
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  .cm,
  .reason {
    font-size: var(--fs-xs);
    white-space: nowrap;
  }
  .reason {
    color: var(--muted);
  }
  .diff {
    font-family: var(--mono);
    font-size: var(--fs-xs);
    white-space: nowrap;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    width: 100%;
    margin-top: var(--sp-3);
    padding: var(--sp-3) 20px;
    border: none;
    border-top: 1px solid var(--border);
    background: none;
    color: var(--muted);
    font-size: var(--fs-xs);
    text-align: left;
  }
  .toggle svg {
    width: 12px;
    height: 12px;
  }
  .toggle svg.open {
    rotate: 90deg;
  }
  .grow {
    flex: 1;
  }
</style>
