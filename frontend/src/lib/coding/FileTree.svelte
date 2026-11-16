<script lang="ts">
  import type { WsChange, WsEntry } from "../api";
  import { changedDirs } from "./coding";

  interface Props {
    title: string;
    /** filhos já carregados de cada pasta ("" é a raiz) */
    entries: Map<string, WsEntry[]>;
    expanded: Set<string>;
    changes: WsChange[];
    activePath: string | null;
    ontoggle: (path: string) => void;
    onopen: (path: string) => void;
  }

  let { title, entries, expanded, changes, activePath, ontoggle, onopen }: Props = $props();

  interface Row {
    entry: WsEntry;
    depth: number;
  }

  const byPath = $derived(new Map(changes.map((c) => [c.path, c])));
  const dirs = $derived(changedDirs(changes.filter((c) => c.status !== "D").map((c) => c.path)));

  const rows = $derived.by(() => {
    const out: Row[] = [];
    const walk = (dir: string, depth: number) => {
      for (const entry of entries.get(dir) ?? []) {
        out.push({ entry, depth });
        if (entry.dir && expanded.has(entry.path)) walk(entry.path, depth + 1);
      }
    };
    walk("", 0);
    return out;
  });

  function tone(e: WsEntry): string {
    if (e.dir) return dirs.has(e.path) ? "mod" : "";
    const s = byPath.get(e.path)?.status;
    return s === "A" || s === "U" ? "add" : s ? "mod" : "";
  }
</script>

<div class="col">
  <div class="head" title={title}>{title}</div>
  <div class="scroll" role="tree" aria-label="Arquivos">
    {#each rows as { entry, depth } (entry.path)}
      {@const st = byPath.get(entry.path)?.status}
      <button
        class="node {tone(entry)}"
        class:on={entry.path === activePath}
        class:ign={entry.ignored}
        role="treeitem"
        aria-selected={entry.path === activePath}
        aria-expanded={entry.dir ? expanded.has(entry.path) : undefined}
        style:padding-left="{8 + depth * 14}px"
        title={entry.path}
        onclick={() => (entry.dir ? ontoggle(entry.path) : onopen(entry.path))}
      >
        <span class="chev" aria-hidden="true">{entry.dir ? (expanded.has(entry.path) ? "▾" : "▸") : ""}</span>
        <span class="name">{entry.name}</span>
        {#if st && !entry.dir}<span class="st {st}">{st}</span>{/if}
      </button>
    {/each}
  </div>
</div>

<style>
  .col {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .head {
    height: 30px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    padding: 0 var(--sp-3);
    font-size: 11px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
    border-bottom: 1px solid var(--border);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--sp-1) 0;
  }
  .node {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    height: 24px;
    padding-right: var(--sp-2);
    text-align: left;
    white-space: nowrap;
    color: var(--text-2);
    font-size: var(--fs-sm);
    background: none;
    border: 0;
    cursor: pointer;
  }
  .node:hover {
    background: var(--surface-2);
  }
  .node.on {
    background: var(--selected);
    color: var(--text);
  }
  .node.ign {
    opacity: 0.5;
  }
  .node.mod .name {
    color: var(--yellow);
  }
  .node.add .name {
    color: var(--green);
  }
  .chev {
    width: 12px;
    flex-shrink: 0;
    text-align: center;
    font-size: 10px;
    color: var(--muted);
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .st {
    margin-left: auto;
    font-family: var(--mono);
    font-size: 11px;
  }
  .st.M,
  .st.R {
    color: var(--yellow);
  }
  .st.A,
  .st.U {
    color: var(--green);
  }
  .st.D {
    color: var(--red);
  }
</style>
