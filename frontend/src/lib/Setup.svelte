<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type Deps, type DeviceCode } from "./api";

  interface Props {
    deps: Deps;
    onrecheck: () => Promise<void>;
  }

  let { deps, onrecheck }: Props = $props();

  let authorizing = $state(false);
  let code = $state<DeviceCode | null>(null);
  let copied = $state(false);
  let error = $state("");

  onMount(() => api.onDeviceCode((d) => (code = d)));

  const needsAuth = $derived(deps.ghInstalled && (!!deps.ghError || deps.missingScopes.length > 0));

  async function authorize(): Promise<void> {
    authorizing = true;
    error = "";
    code = null;
    try {
      await api.authorizeGitHub();
      await onrecheck();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      authorizing = false;
      code = null;
    }
  }

  async function copy(): Promise<void> {
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code.code);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {
      /* sem acesso à área de transferência: o código continua visível para digitar */
    }
  }
</script>

<div class="setup">
  <h3>Falta pouco para decolar</h3>
  <ol>
    <li class:done={deps.ghInstalled}>
      <strong>GitHub CLI</strong>
      {#if deps.ghInstalled}
        <span>instalado</span>
      {:else}
        <span>
          O app usa o <code>gh</code> para ler PRs e publicar reviews na sua conta.
          <button class="link" onclick={() => api.openURL("https://cli.github.com")}>Instalar o gh</button>
        </span>
      {/if}
    </li>
    <li class:done={deps.ghInstalled && !needsAuth}>
      <strong>Acesso ao GitHub</strong>
      {#if !deps.ghInstalled}
        <span>depois de instalar o gh</span>
      {:else if !needsAuth}
        <span>autorizado</span>
      {:else}
        <span>
          {deps.ghError ? "Conecte sua conta" : `Faltam as permissões ${deps.missingScopes.join(", ")}`}:
          ler PRs (inclusive privadas), comentar e aprovar. Tudo sai na sua conta, só quando você clicar em enviar.
        </span>
        {#if code}
          <div class="code">
            <span>Digite este código no GitHub:</span>
            <button class="code-value" onclick={copy} title="Copiar">{code.code}</button>
            <span class="muted">{copied ? "copiado!" : "clique para copiar"}</span>
            <button class="link" onclick={() => code && api.openURL(code.url)}>Abrir a página do GitHub</button>
          </div>
        {/if}
        <div class="actions">
          <button class="btn primary" onclick={authorize} disabled={authorizing}>
            {authorizing ? "Aguardando o GitHub..." : deps.ghError ? "Conectar ao GitHub" : "Autorizar permissões"}
          </button>
          {#if authorizing}<button class="btn" onclick={() => api.cancelAuthorize()}>Cancelar</button>{/if}
        </div>
      {/if}
    </li>
    <li class:done={!deps.claudeError}>
      <strong>Claude Code</strong>
      {#if !deps.claudeError}
        <span>instalado (usa o login que você já tem, sem chave de API)</span>
      {:else}
        <span>
          O revisor roda no seu Claude Code. Instale, abra <code>claude</code> no terminal uma vez e faça login.
          <button class="link" onclick={() => api.openURL("https://claude.com/claude-code")}>Instalar o Claude Code</button>
        </span>
      {/if}
    </li>
  </ol>
  {#if error}<p class="error">{error}</p>{/if}
  <button class="btn" onclick={onrecheck}>Verificar de novo</button>
</div>

<style>
  .setup {
    margin-top: 18px;
    width: 100%;
    padding: 16px 18px;
    border-radius: 14px;
    background: rgba(227, 179, 65, 0.07);
    border: 1px solid rgba(227, 179, 65, 0.3);
    text-align: left;
    font-size: 13px;
  }
  h3 {
    margin: 0 0 10px;
    font-size: 14px;
    color: #f6d68a;
  }
  ol {
    margin: 0 0 12px;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  li span {
    color: var(--muted);
    margin-left: 6px;
  }
  li.done strong::after {
    content: " ✓";
    color: var(--green);
  }
  code {
    font: 12px var(--mono);
    color: var(--claude-soft);
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    font-size: 13px;
    padding: 0;
    margin-left: 4px;
  }
  .code {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
    margin: 10px 0 0;
  }
  .code span {
    margin: 0;
  }
  .code-value {
    font: 700 20px var(--mono);
    letter-spacing: 0.12em;
    color: var(--text);
    background: rgba(0, 0, 0, 0.35);
    border: 1px dashed var(--border);
    border-radius: 8px;
    padding: 4px 12px;
  }
  .muted {
    font-size: 12px;
  }
  .actions {
    display: flex;
    gap: 8px;
    margin-top: 10px;
  }
  .error {
    color: #ff9b94;
    white-space: pre-wrap;
  }
</style>
