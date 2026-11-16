<script lang="ts">
  import type { WsChanges } from "../api";
  import { baseName, dirName } from "./coding";

  interface Props {
    changes: WsChanges | null;
    activeDiff: string | null;
    onopen: (path: string) => void;
  }

  let { changes, activeDiff, onopen }: Props = $props();

  const files = $derived(changes?.files ?? []);
  const added = $derived(files.reduce((s, f) => s + f.added, 0));
  const removed = $derived(files.reduce((s, f) => s + f.removed, 0));
</script>

<div class="col">
  <div class="head">Alterações<span class="count">{files.length}</span></div>
  {#if changes && !changes.isRepo}
    <p class="empty">Esta pasta não é um repositório git.</p>
  {:else}
    <div class="sum">
      <span class="branch" title="Branch atual">⎇ {changes?.branch || "…"}</span>
      <span class="nums"><span class="plus">+{added}</span> <span class="minus">−{removed}</span></span>
    </div>
    <div class="scroll">
      {#each files as f (f.path)}
        <button class="chg" class:on={activeDiff === f.path} title={f.path} onclick={() => onopen(f.path)}>
          <span class="st {f.status}">{f.status}</span>
          <span class="p">{baseName(f.path)}<small>{dirName(f.path)}</small></span>
          <span class="nums">
            {#if f.added}<span class="plus">+{f.added}</span>{/if}
            {#if f.removed}<span class="minus">−{f.removed}</span>{/if}
          </span>
        </button>
      {:else}
        {#if changes}<p class="empty">Nada alterado desde o último commit.</p>{/if}
      {/each}
    </div>
  {/if}
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
  }
  .count {
    margin-left: auto;
    font-family: var(--mono);
    letter-spacing: 0;
  }
  .sum {
    display: flex;
    align-items: baseline;
    gap: var(--sp-3);
    padding: 10px var(--sp-3);
    border-bottom: 1px solid var(--border);
    font-size: var(--fs-xs);
  }
  .branch {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-2);
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sum .nums {
    margin-left: auto;
    font-size: var(--fs-sm);
  }
  .scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--sp-1) 0;
  }
  .chg {
    display: grid;
    grid-template-columns: 16px minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--sp-2);
    width: 100%;
    padding: var(--sp-1) var(--sp-3);
    text-align: left;
    color: var(--text-2);
    font-size: var(--fs-sm);
    background: none;
    border: 0;
    cursor: pointer;
  }
  .chg:hover {
    background: var(--surface-2);
  }
  .chg.on {
    background: var(--selected);
    color: var(--text);
  }
  .p {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .p small {
    margin-left: 6px;
    font-size: 11px;
    color: var(--muted);
  }
  .nums {
    font-family: var(--mono);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
    display: flex;
    gap: 4px;
  }
  .st {
    font-family: var(--mono);
    font-size: 11px;
    text-align: center;
  }
  .st.M,
  .st.R {
    color: var(--yellow);
  }
  .st.A,
  .st.U,
  .plus {
    color: var(--green);
  }
  .st.D,
  .minus {
    color: var(--red);
  }
  .empty {
    margin: 0;
    padding: var(--sp-3);
    color: var(--muted);
    font-size: var(--fs-xs);
  }
</style>
