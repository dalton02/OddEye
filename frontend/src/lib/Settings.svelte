<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type Config, type RepoConfig, type RepoOption } from "./api";

  interface Props {
    onclose: () => void;
  }

  let { onclose }: Props = $props();

  let config = $state<Config | null>(null);
  let defaultPrompt = $state("");
  let configDir = $state("");
  let prompt = $state("");
  let error = $state("");
  let saving = $state(false);
  let saved = $state(false);

  let repoOptions = $state<RepoOption[] | null>(null);
  let repoError = $state("");
  let query = $state("");
  let expanded = $state<string | null>(null);

  const intervals = [2, 5, 10, 15, 30, 60];

  let version = $state("");
  let updateStatus = $state("");
  let checkingUpdate = $state(false);

  /* se houver versão nova, o backend emite update:available e o App abre o modal */
  async function checkUpdate(): Promise<void> {
    checkingUpdate = true;
    updateStatus = "";
    try {
      const u = await api.checkUpdate();
      updateStatus = u ? `Versão ${u.version} disponível.` : "Você já está na versão mais recente.";
    } catch (e) {
      updateStatus = errorMessage(e);
    } finally {
      checkingUpdate = false;
    }
  }

  onMount(async () => {
    api.version().then((v) => (version = v));
    try {
      const s = await api.getSettings();
      config = s.config;
      defaultPrompt = s.defaultPrompt;
      configDir = s.configDir;
      prompt = s.config.reviewerPrompt || s.defaultPrompt;
    } catch (e) {
      error = errorMessage(e);
    }
    api
      .listRepos()
      .then((r) => (repoOptions = r))
      .catch((e) => (repoError = errorMessage(e)));
  });

  const key = (r: { owner: string; name: string }): string => `${r.owner}/${r.name}`.toLowerCase();

  const suggestions = $derived.by(() => {
    if (!config || !repoOptions) return [];
    const added = new Set(config.repos.map(key));
    const q = query.trim().toLowerCase();
    return repoOptions
      .filter((r) => !added.has(r.fullName.toLowerCase()) && (!q || r.fullName.toLowerCase().includes(q)))
      .slice(0, 8);
  });

  const typedRepo = $derived.by(() => {
    const m = query.trim().match(/^(?:https?:\/\/github\.com\/)?([\w.-]+)\/([\w.-]+?)(?:\.git)?\/?$/);
    return m ? { owner: m[1], name: m[2] } : null;
  });

  function addRepo(fullName: string): void {
    if (!config) return;
    const [owner, name] = fullName.split("/");
    if (!owner || !name || config.repos.some((r) => key(r) === `${owner}/${name}`.toLowerCase())) return;
    config.repos = [
      ...config.repos,
      { owner, name, onlyReviewRequested: false, preAnalyze: false, promptOverride: "" }
    ];
    query = "";
  }

  function removeRepo(r: RepoConfig): void {
    if (!config) return;
    config.repos = config.repos.filter((x) => key(x) !== key(r));
  }

  function onSearchKey(e: KeyboardEvent): void {
    if (e.key !== "Enter") return;
    e.preventDefault();
    if (suggestions.length > 0) addRepo(suggestions[0].fullName);
    else if (typedRepo) addRepo(`${typedRepo.owner}/${typedRepo.name}`);
  }

  async function save(): Promise<void> {
    if (!config) return;
    saving = true;
    error = "";
    try {
      const toSave: Config = { ...config, reviewerPrompt: prompt.trim() === defaultPrompt.trim() ? "" : prompt };
      config = await api.saveSettings(toSave);
      saved = true;
      setTimeout(onclose, 600);
    } catch (e) {
      error = errorMessage(e);
    } finally {
      saving = false;
    }
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !saving) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !saving && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="settings-title">
    <header>
      <h2 id="settings-title">Configurações</h2>
      <button class="close" onclick={onclose} aria-label="Fechar">×</button>
    </header>

    {#if !config}
      <p class="muted">{error || "Carregando..."}</p>
    {:else}
      <section>
        <h3>Repositórios vigiados</h3>
        <p class="muted">O Oddeye confere esses repositórios de tempos em tempos e avisa quando chega PR nova.</p>

        <div class="picker">
          <input
            bind:value={query}
            onkeydown={onSearchKey}
            placeholder={repoOptions ? "Buscar repositório ou digitar dono/repo" : "Carregando seus repositórios..."}
            spellcheck="false"
            autocomplete="off"
          />
          {#if query.trim() && (suggestions.length || typedRepo)}
            <ul class="suggestions">
              {#each suggestions as s (s.fullName)}
                <li>
                  <button onclick={() => addRepo(s.fullName)}>
                    {s.fullName}
                    {#if s.private}<span class="tag">privado</span>{/if}
                  </button>
                </li>
              {/each}
              {#if suggestions.length === 0 && typedRepo}
                <li>
                  <button onclick={() => typedRepo && addRepo(`${typedRepo.owner}/${typedRepo.name}`)}>
                    Adicionar {typedRepo.owner}/{typedRepo.name}
                  </button>
                </li>
              {/if}
            </ul>
          {/if}
        </div>
        {#if repoError}<p class="warn">Não consegui listar seus repositórios ({repoError}). Digite dono/repo.</p>{/if}

        {#if config.repos.length === 0}
          <p class="empty">Nenhum repositório ainda.</p>
        {:else}
          <ul class="repos">
            {#each config.repos as r (key(r))}
              <li>
                <div class="repo-head">
                  <span class="name">{r.owner}/{r.name}</span>
                  <button class="link" onclick={() => (expanded = expanded === key(r) ? null : key(r))}>
                    {expanded === key(r) ? "Fechar" : "Opções"}
                  </button>
                  <button class="link danger" onclick={() => removeRepo(r)}>Remover</button>
                </div>
                {#if expanded === key(r)}
                  <div class="repo-opts">
                    <label class="check">
                      <input type="checkbox" bind:checked={r.onlyReviewRequested} />
                      <span>Só avisar quando pedirem o meu review</span>
                    </label>
                    <label class="check">
                      <input type="checkbox" bind:checked={r.preAnalyze} />
                      <span>
                        Pré-analisar com o Claude ao chegar
                        <small>Abrir a PR fica instantâneo, mas cada PR nova gasta uma análise da sua assinatura.</small>
                      </span>
                    </label>
                    <label class="stack">
                      <span>Instruções próprias para este repositório <small>(vazio = usa as globais)</small></span>
                      <textarea rows="5" bind:value={r.promptOverride} placeholder="Ex.: este repo é um SDK público; priorize compatibilidade de API."></textarea>
                    </label>
                  </div>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <section>
        <h3>Notificações</h3>
        <div class="grid">
          <label class="stack">
            <span>Verificar a cada</span>
            <select bind:value={config.pollIntervalMinutes}>
              {#each intervals as m}<option value={m}>{m} minutos</option>{/each}
            </select>
          </label>
        </div>
        <label class="check"><input type="checkbox" bind:checked={config.includeDrafts} /><span>Avisar também de PRs em rascunho</span></label>
        <label class="check"><input type="checkbox" bind:checked={config.notifyOwnPRs} /><span>Avisar das minhas próprias PRs</span></label>
        <label class="check"><input type="checkbox" bind:checked={config.launchAtLogin} /><span>Abrir o Oddeye ao entrar no sistema (direto na bandeja)</span></label>
      </section>

      <section>
        <h3>Revisor (Claude)</h3>
        <label class="stack">
          <span>Modelo <small>(vazio = padrão do seu Claude Code)</small></span>
          <input list="models" bind:value={config.model} placeholder="padrão" spellcheck="false" />
          <datalist id="models"><option value="sonnet"></option><option value="opus"></option><option value="haiku"></option></datalist>
        </label>
        <label class="stack">
          <span class="row">
            Instruções do revisor
            <button class="link" onclick={() => (prompt = defaultPrompt)} disabled={prompt === defaultPrompt}>Restaurar padrão</button>
          </span>
          <textarea class="prompt" rows="12" bind:value={prompt} spellcheck="false"></textarea>
        </label>
        <p class="muted small">
          Diga o que o revisor deve priorizar, o tom e o idioma. As regras de formato da resposta (caminhos, linhas,
          JSON) são do app e entram sempre, então nada aqui quebra o tour. Trocar as instruções gera uma análise nova.
        </p>
      </section>

      <section class="version">
        <span>Oddeye {version === "dev" ? "(desenvolvimento)" : `v${version}`}</span>
        <button class="btn" onclick={checkUpdate} disabled={checkingUpdate || version === "dev"}>
          {checkingUpdate ? "Procurando..." : "Procurar atualizações"}
        </button>
        {#if updateStatus}<span class="muted small">{updateStatus}</span>{/if}
      </section>

      {#if error}<p class="error">{error}</p>{/if}

      <footer>
        <span class="muted small">Dados em {configDir}</span>
        <button class="btn" onclick={onclose} disabled={saving}>Cancelar</button>
        <button class="btn primary" onclick={save} disabled={saving}>{saved ? "Salvo!" : saving ? "Salvando..." : "Salvar"}</button>
      </footer>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: rgba(2, 3, 10, 0.82);
    display: grid;
    place-items: center;
    padding: 24px;
  }
  .modal {
    width: min(720px, 100%);
    max-height: 90vh;
    overflow: auto;
    background: var(--panel-strong);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 30px 90px rgba(0, 0, 0, 0.6);
    text-align: left;
    /* sem isso o WebKitGTK desenha select e checkbox com o tema claro do sistema */
    color-scheme: dark;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  h2 {
    margin: 0;
    font-size: 20px;
  }
  .close {
    background: none;
    border: none;
    color: var(--muted);
    font-size: 24px;
    line-height: 1;
  }
  h3 {
    margin: 0 0 6px;
    font-size: 13px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  section {
    padding: 16px 0;
    border-bottom: 1px solid var(--border);
  }
  .muted {
    color: var(--muted);
    font-size: 13px;
    margin: 0 0 10px;
  }
  .small {
    font-size: 12px;
  }
  small {
    display: block;
    color: var(--muted);
    font-size: 12px;
  }
  .stack small {
    display: inline;
  }
  .warn {
    color: var(--yellow);
    font-size: 12px;
  }
  .error {
    color: #ff9b94;
    font-size: 13px;
    white-space: pre-wrap;
  }
  .empty {
    color: var(--muted);
    font-size: 13px;
    font-style: italic;
  }
  input:not([type="checkbox"]),
  select,
  textarea {
    width: 100%;
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid var(--border);
    border-radius: 10px;
    color: var(--text);
    padding: 9px 12px;
    font: 13px var(--ui);
    outline: none;
  }
  input:focus,
  select:focus,
  textarea:focus {
    border-color: rgba(140, 160, 255, 0.45);
  }
  select {
    appearance: none;
    padding-right: 32px;
    background-image: linear-gradient(45deg, transparent 50%, var(--muted) 50%),
      linear-gradient(135deg, var(--muted) 50%, transparent 50%);
    background-position: calc(100% - 17px) 50%, calc(100% - 12px) 50%;
    background-size: 5px 5px;
    background-repeat: no-repeat;
  }
  select option {
    background: #15161c;
    color: var(--text);
  }
  textarea {
    resize: vertical;
    line-height: 1.5;
  }
  textarea.prompt {
    font: 12.5px/1.55 var(--mono);
  }
  .picker {
    position: relative;
  }
  .suggestions {
    position: absolute;
    left: 0;
    right: 0;
    top: calc(100% + 4px);
    z-index: 2;
    list-style: none;
    margin: 0;
    padding: 4px;
    background: #15161c;
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.5);
  }
  .suggestions button {
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    color: var(--text);
    padding: 7px 10px;
    border-radius: 7px;
    font: 13px var(--mono);
  }
  .suggestions button:hover {
    background: rgba(255, 255, 255, 0.06);
  }
  .tag {
    margin-left: 6px;
    font: 11px var(--ui);
    color: var(--muted);
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 1px 7px;
  }
  .repos {
    list-style: none;
    margin: 12px 0 0;
    padding: 0;
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .repos li + li {
    border-top: 1px solid var(--border);
  }
  .repo-head {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 9px 12px;
  }
  .name {
    flex: 1;
    font: 13px var(--mono);
  }
  .repo-opts {
    padding: 4px 12px 12px;
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    font-size: 12px;
    padding: 0;
  }
  .link:disabled {
    color: var(--muted);
    opacity: 0.6;
  }
  .link.danger {
    color: #ff9b94;
  }
  .check {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    font-size: 13px;
    margin: 10px 0 0;
    cursor: pointer;
  }
  .check input {
    appearance: none;
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    margin: 1px 0 0;
    border: 1px solid rgba(160, 170, 210, 0.35);
    border-radius: 4px;
    background: rgba(0, 0, 0, 0.35);
    display: grid;
    place-items: center;
    cursor: pointer;
  }
  .check input:checked {
    background: var(--claude);
    border-color: var(--claude);
  }
  .check input:checked::after {
    content: "";
    width: 4px;
    height: 8px;
    margin-top: -2px;
    border: solid #fff;
    border-width: 0 2px 2px 0;
    transform: rotate(45deg);
  }
  .stack {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 13px;
    margin-top: 10px;
  }
  .row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .grid {
    display: grid;
    grid-template-columns: minmax(0, 220px);
  }
  .version {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    font-size: 13px;
  }
  .version .muted {
    margin: 0;
  }
  footer {
    display: flex;
    align-items: center;
    gap: 10px;
    padding-top: 16px;
  }
  footer .muted {
    flex: 1;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
