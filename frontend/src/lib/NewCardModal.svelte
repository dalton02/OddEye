<script lang="ts">
  import { api, errorMessage, type CreatedCard } from "./api";
  import { branchName, type Board } from "./board";

  interface Props {
    board: Board;
    statusId: string; // coluna onde o card nasce ("" = sem status)
    repos: string[];
    lastRepo: string;
    viewer: string;
    oncreated: (res: CreatedCard) => void;
    onclose: () => void;
  }

  let { board, statusId, repos, lastRepo, viewer, oncreated, onclose }: Props = $props();

  // svelte-ignore state_referenced_locally
  let status = $state(statusId);
  // svelte-ignore state_referenced_locally
  let repo = $state(repos.includes(lastRepo) ? lastRepo : (repos[0] ?? ""));
  let title = $state("");
  let body = $state("");
  // svelte-ignore state_referenced_locally
  let typeId = $state(board.type?.options.find((o) => /task|tarefa/i.test(o.name))?.id ?? "");
  // svelte-ignore state_referenced_locally
  let assignMe = $state(!!viewer);
  let withBranch = $state(false);
  let sending = $state(false);
  let error = $state("");

  const typeName = $derived(board.type?.options.find((o) => o.id === typeId)?.name ?? "");
  const statusName = $derived(board.status.options.find((o) => o.id === status)?.name ?? "Sem status");
  const canSend = $derived(title.trim() !== "" && repo.trim() !== "" && !sending);
  // o número só existe depois de criar a issue: a prévia mostra <nº> e o Go recebe o nome sem ele
  const preview = $derived(branchName(typeName, null, title).replace("/", "/<nº>-").replace(/-$/, ""));

  async function send(): Promise<void> {
    if (!canSend) return;
    sending = true;
    error = "";
    try {
      const res = await api.createCard({
        projectId: board.projectId,
        repo: repo.trim(),
        title: title.trim(),
        body,
        assignMe,
        statusFieldId: board.status.id,
        statusOptionId: status,
        typeFieldId: board.type?.id ?? "",
        typeOptionId: typeId,
      });
      if (withBranch && !res.warning) {
        try {
          const d = await api.issueDetail(res.card.contentId);
          const base = d.branches.find((b) => b.name === d.defaultBranch);
          if (!base) throw new Error("o repositório não tem branch padrão");
          await api.createIssueBranch(d.id, d.repoId, branchName(typeName, res.card.number, res.card.title), base.oid);
        } catch (e) {
          res.warning = `card #${res.card.number} criado, mas a branch não: ${errorMessage(e)}`;
        }
      }
      oncreated(res);
    } catch (e) {
      error = errorMessage(e);
      sending = false;
    }
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !sending) onclose();
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      void send();
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !sending && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="newcard-title">
    <form
      class="contents"
      onsubmit={(e) => {
        e.preventDefault();
        void send();
      }}
    >
      <div class="modal-head">
        <h2 id="newcard-title">Novo card</h2>
        <p>{board.title}</p>
      </div>

      <div class="modal-body">
        <label class="block">
          <span class="label">Título</span>
          <!-- svelte-ignore a11y_autofocus -->
          <input class="field" bind:value={title} autofocus placeholder="O que precisa ser feito" />
        </label>

        <div class="grid">
          <label class="block">
            <span class="label">Repositório</span>
            <input class="field mono" bind:value={repo} list="board-repos" placeholder="dono/repositório" spellcheck="false" />
            <datalist id="board-repos">
              {#each repos as r (r)}<option value={r}></option>{/each}
            </datalist>
          </label>
          <label class="block">
            <span class="label">Coluna</span>
            <select class="field" bind:value={status}>
              <option value="">Sem status</option>
              {#each board.status.options as o (o.id)}<option value={o.id}>{o.name}</option>{/each}
            </select>
          </label>
        </div>

        {#if board.type}
          <div class="block">
            <span class="label">{board.type.name}</span>
            <div class="seg" role="group" aria-label={board.type.name}>
              {#each board.type.options as o (o.id)}
                <button type="button" class:on={typeId === o.id} aria-pressed={typeId === o.id} onclick={() => (typeId = typeId === o.id ? "" : o.id)}>
                  {o.name}
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <label class="block">
          <span class="label">Descrição (opcional)</span>
          <textarea class="field" bind:value={body} rows="4" placeholder="Contexto, critérios de aceite, links. Markdown funciona."></textarea>
        </label>

        <div class="opts">
          {#if viewer}
            <label class="check"><input type="checkbox" bind:checked={assignMe} /> Atribuir a mim ({viewer})</label>
          {/if}
          <label class="check"><input type="checkbox" bind:checked={withBranch} /> Criar a branch remota junto, ligada à issue</label>
          {#if withBranch}
            <span class="mono preview">{preview || "<tipo>/<nº>-<título>"}</span>
          {/if}
        </div>

        {#if error}<p class="error-text">{error}</p>{/if}
      </div>

      <div class="modal-foot">
        <span class="hint">Entra em {statusName} · Ctrl+Enter cria</span>
        <button type="button" class="btn" disabled={sending} onclick={onclose}>Cancelar</button>
        <button type="submit" class="btn primary" disabled={!canSend}>{sending ? "Criando..." : "Criar card"}</button>
      </div>
    </form>
  </div>
</div>

<style>
  .modal {
    max-width: 560px;
  }
  .contents {
    display: contents;
  }
  .block {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .modal-body {
    gap: var(--sp-4);
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--sp-3);
  }
  .seg {
    align-self: flex-start;
    flex-wrap: wrap;
  }
  .opts {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    padding: var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface-2);
  }
  .check {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    cursor: pointer;
  }
  .preview {
    padding-left: 24px;
    color: var(--text-2);
    font-size: var(--fs-xs);
    overflow-wrap: anywhere;
  }
  .hint {
    margin-right: auto;
    color: var(--muted);
    font-size: var(--fs-xs);
  }
</style>
