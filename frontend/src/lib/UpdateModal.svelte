<script lang="ts">
  import { api, type UpdateInfo, type UpdateProgress } from "./api";

  interface Props {
    info: UpdateInfo;
    onclose: () => void;
  }

  let { info, onclose }: Props = $props();

  let progress = $state<UpdateProgress | null>(null);
  const busy = $derived(!!progress && progress.stage !== "error");

  const percent = $derived(
    progress?.stage === "downloading" && progress.total > 0 ? Math.min(100, Math.round((progress.done / progress.total) * 100)) : null
  );

  const status = $derived.by((): string => {
    switch (progress?.stage) {
      case "downloading":
        return percent === null ? "Baixando…" : `Baixando… ${percent}%`;
      case "installing":
        return info.needsPassword ? "Instalando… confirme a senha na janela do sistema." : "Instalando…";
      case "restarting":
        return "Reiniciando…";
      default:
        return "";
    }
  });

  $effect(() => api.onUpdateProgress((p) => (progress = p)));

  async function install(): Promise<void> {
    progress = { stage: "downloading", done: 0, total: 0 };
    try {
      await api.installUpdate();
    } catch (e) {
      /* o backend já emitiu o evento de erro com os detalhes; isto cobre falha da própria chamada */
      if (progress?.stage !== "error") progress = { stage: "error", done: 0, total: 0, error: String(e) };
    }
  }

  async function skip(): Promise<void> {
    await api.skipUpdate(info.version);
    onclose();
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !busy) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !busy && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="update-title">
    <div class="modal-head">
      <h2 id="update-title">Nova versão disponível: v{info.version}</h2>
      <p>Você está na v{info.current}.</p>
    </div>

    <div class="modal-body">
      {#if info.notes}
        <div class="block">
          <span class="label">O que mudou</span>
          <p class="notes">{info.notes}</p>
        </div>
      {/if}

      {#if progress && progress.stage !== "error"}
        <div class="block progress" role="status">
          <div class="bar"><div class="fill" class:indeterminate={percent === null} style:width={percent === null ? "40%" : `${percent}%`}></div></div>
          <p class="muted">{status}</p>
        </div>
      {:else if progress?.stage === "error"}
        <p class="error-text">
          {progress.manual
            ? "Não consegui atualizar sozinho neste computador. Baixe a versão nova pela página do release."
            : `A atualização falhou: ${progress.error}`}
        </p>
      {:else if !info.automatic}
        <p class="muted">Este jeito de instalação não atualiza sozinho. Baixe a versão nova pela página do release.</p>
      {:else if info.needsPassword}
        <p class="muted">O sistema vai pedir sua senha uma vez para instalar o pacote. Depois o Oddeye reinicia sozinho.</p>
      {:else}
        <p class="muted">O Oddeye baixa a versão nova, instala e reinicia sozinho. Seus rascunhos ficam salvos.</p>
      {/if}
    </div>

    {#if !busy}
      <div class="modal-foot">
        <button class="link quiet" onclick={skip}>Pular esta versão</button>
        <span class="spacer"></span>
        <button class="btn" onclick={onclose}>Depois</button>
        {#if info.automatic && !progress?.manual}
          <button class="btn primary" onclick={install}>{progress?.stage === "error" ? "Tentar de novo" : "Atualizar agora"}</button>
        {/if}
        {#if !info.automatic || progress?.stage === "error"}
          <button class="btn" class:primary={!info.automatic || progress?.manual} onclick={() => api.openReleasePage()}>Baixar manualmente</button>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  /* fica acima do modal de review */
  .backdrop {
    z-index: calc(var(--z-modal) + 1);
  }
  .modal {
    max-width: 520px;
  }
  .modal-body {
    gap: var(--sp-4);
  }
  .modal-body p {
    margin: 0;
  }
  .block {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .notes {
    max-height: 220px;
    overflow: auto;
    white-space: pre-wrap;
  }
  .muted {
    color: var(--muted);
  }
  .bar {
    height: 4px;
    border-radius: var(--r-sm);
    background: var(--selected);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    background: var(--text);
  }
  /* única animação do app: sem total conhecido, a barra precisa mostrar que o download segue vivo */
  .fill.indeterminate {
    animation: slide 1.2s ease-in-out infinite;
  }
  @keyframes slide {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(250%);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .fill.indeterminate {
      animation: none;
    }
  }
  .spacer {
    flex: 1;
  }
</style>
