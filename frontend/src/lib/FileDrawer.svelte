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
    <button class="close" onclick={onclose} aria-label="Fechar">×</button>
  </header>

  <div class="scroll">
    <h3>Selecionados pelo Claude <span>{session.files.length}</span></h3>
    {#each session.files as f, i (f.path)}
      <button class="item" class:active={activePath === f.path} onclick={() => onopen(f.path)}>
        <span class="num" class:seen={visited.has(f.path)}>{i + 1}</span>
        <span class="info">
          <span class="name">{name(f.path)}</span>
          <span class="dir">{dir(f.path)}</span>
        </span>
        <span class="right">
          {#if countFor(f.path)}<span class="cm">{countFor(f.path)}</span>{/if}
          <span class="sev {f.severity}">{severityLabel[f.severity]}</span>
        </span>
      </button>
    {/each}

    {#if session.others.length}
      <h3>Outros alterados <span>{session.others.length}</span></h3>
      {#each session.others as f (f.path)}
        <button class="item" class:active={activePath === f.path} onclick={() => onopen(f.path)}>
          <span class="info">
            <span class="name">{name(f.path)}</span>
            <span class="dir">{dir(f.path)}</span>
          </span>
          <span class="right">
            {#if countFor(f.path)}<span class="cm">{countFor(f.path)}</span>{/if}
            <span class="diff"><b>+{f.additions}</b> <i>-{f.deletions}</i></span>
          </span>
        </button>
      {/each}
    {/if}

    {#if session.skipped.length}
      <h3>
        <button class="toggle" onclick={() => (showSkipped = !showSkipped)}>
          {showSkipped ? "▾" : "▸"} Descartados (testes, docs, gerados)
        </button>
        <span>{session.skipped.length}</span>
      </h3>
      {#if showSkipped}
        {#each session.skipped as s (s.path)}
          <button class="item dim" class:active={activePath === s.path} onclick={() => onopen(s.path)}>
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
    z-index: 40;
    background: var(--panel-strong);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    box-shadow: 20px 0 60px rgba(0, 0, 0, 0.5);
    animation: slide 0.2s ease-out;
  }
  @keyframes slide {
    from {
      transform: translateX(-30px);
      opacity: 0;
    }
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 18px;
    border-bottom: 1px solid var(--border);
  }
  h2 {
    margin: 0;
    font-size: 16px;
  }
  .close {
    background: none;
    border: none;
    color: var(--muted);
    font-size: 24px;
    line-height: 1;
  }
  .scroll {
    overflow: auto;
    padding: 8px 10px 20px;
  }
  h3 {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 16px 8px 6px;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--muted);
  }
  h3 span {
    font-family: var(--mono);
  }
  .toggle {
    background: none;
    border: none;
    color: inherit;
    font: inherit;
    letter-spacing: inherit;
    text-transform: inherit;
    padding: 0;
  }
  .item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 10px;
    border-radius: 10px;
    background: none;
    border: 1px solid transparent;
    color: var(--text);
    text-align: left;
  }
  .item:hover {
    background: rgba(255, 255, 255, 0.05);
  }
  .item.active {
    background: rgba(217, 119, 87, 0.12);
    border-color: rgba(217, 119, 87, 0.4);
  }
  .item.dim {
    opacity: 0.65;
  }
  .num {
    width: 22px;
    height: 22px;
    flex-shrink: 0;
    border-radius: 50%;
    border: 1px solid var(--border);
    display: grid;
    place-items: center;
    font-size: 11px;
    color: var(--muted);
  }
  .num.seen {
    background: var(--claude);
    border-color: var(--claude);
    color: #fff;
  }
  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .name {
    font-family: var(--mono);
    font-size: 12.5px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .dir {
    font-size: 11px;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  .right {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .cm {
    background: var(--claude);
    color: #fff;
    font-size: 10px;
    font-weight: 800;
    border-radius: 999px;
    padding: 1px 6px;
  }
  .diff {
    font-family: var(--mono);
    font-size: 11px;
  }
  .diff b {
    color: var(--green);
    font-weight: 400;
  }
  .diff i {
    color: var(--red);
    font-style: normal;
  }
  .reason {
    font-size: 11px;
    color: var(--muted);
  }
</style>
