<script lang="ts">
  import { api } from "./api";
  import { player } from "./player.svelte";

  let open = $state(false);
</script>

{#if player.unavailable}
  <div class="player off" class:open>
    <button class="title" onclick={() => (open = !open)} title="Por que o som está desligado">
      <svg class="mute" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9zM17 9l5 6M22 9l-5 6" /></svg>
      <span class="name">Som indisponível</span>
    </button>
    {#if open}
      <div class="panel" role="dialog" aria-label="Som indisponível">
        <p class="hint">O sistema não tem o que o app precisa para tocar som. Falta:</p>
        <ul class="missing">
          {#each player.unavailable.missing as m (m)}<li>{m}</li>{/each}
        </ul>
        {#if player.unavailable.hint}
          <p class="hint">Instale com o comando abaixo e abra o Oddeye de novo:</p>
          <code class="cmd">{player.unavailable.hint}</code>
        {:else}
          <p class="hint">Instale o pacote de plugins "good" do GStreamer 1.0 da sua distribuição e abra o Oddeye de novo.</p>
        {/if}
        <p class="hint">Todo o resto do app funciona normalmente sem som.</p>
      </div>
    {/if}
  </div>
{:else}
<div class="player" class:open>
  <button class="icon" onclick={() => player.prev()} title="Anterior" aria-label="Anterior">
    <svg viewBox="0 0 24 24"><path d="M6 5v14M19 5l-10 7 10 7z" /></svg>
  </button>
  <button class="icon main" onclick={() => player.toggle()} title={player.playing ? "Pausar" : "Tocar"} aria-label={player.playing ? "Pausar" : "Tocar"}>
    {#if player.playing}
      <svg viewBox="0 0 24 24"><path d="M8 5v14M16 5v14" /></svg>
    {:else}
      <svg viewBox="0 0 24 24"><path d="M7 5l12 7-12 7z" /></svg>
    {/if}
  </button>
  <button class="icon" onclick={() => player.next()} title="Próxima" aria-label="Próxima">
    <svg viewBox="0 0 24 24"><path d="M18 5v14M5 5l10 7-10 7z" /></svg>
  </button>
  <button
    class="title"
    onclick={() => {
      open = !open;
      if (open) void player.reload();
    }}
    title="Opções de som"
  >
    {#if player.playing}<span class="eq"><i></i><i></i><i></i></span>{/if}
    <span class="name">{player.title}</span>
  </button>

  {#if open}
    <div class="panel" role="dialog" aria-label="Opções de som">
      <label class="row">
        <span>Volume</span>
        <input type="range" min="0" max="1" step="0.05" value={player.volume} oninput={(e) => player.setVolume(Number(e.currentTarget.value))} />
      </label>
      <label class="row check">
        <input type="checkbox" checked={player.sfx} onchange={() => player.toggleSfx()} />
        <span>Som suave ao clicar nos botões</span>
      </label>
      {#if player.error}<p class="err">{player.error}</p>{/if}
      <div class="lib">
        <p class="hint">
          {#if player.mode === "ambient"}
            Tocando a trilha ambiente. Coloque suas músicas (mp3, ogg, flac, wav, m4a) na pasta abaixo e elas tocam em ordem, em loop.
          {:else}
            {player.library.tracks.length} música{player.library.tracks.length === 1 ? "" : "s"} na pasta, tocando em ordem.
          {/if}
        </p>
        <code title={player.library.dir}>{player.library.dir}</code>
        <div class="actions">
          <button class="btn" onclick={() => api.openMusicFolder()}>Abrir pasta</button>
          <button class="btn" onclick={() => player.chooseFolder()}>Escolher outra</button>
          <button class="btn" onclick={() => player.reload()}>Recarregar</button>
        </div>
      </div>
    </div>
  {/if}
</div>
{/if}

<style>
  .mute {
    width: 15px;
    height: 15px;
    flex-shrink: 0;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .missing {
    margin: 4px 0 8px;
    padding-left: 18px;
    font-size: 12.5px;
    line-height: 1.5;
  }
  .cmd {
    display: block;
    user-select: all;
    white-space: normal;
    word-break: break-word;
    margin-bottom: 8px;
  }

  .player {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 2px;
    height: 34px;
    padding: 0 6px 0 4px;
    border-radius: 999px;
    background: var(--panel);
    border: 1px solid var(--border);
    max-width: 320px;
  }
  .icon {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: none;
    background: none;
    display: grid;
    place-items: center;
    padding: 0;
  }
  .icon:hover {
    background: rgba(255, 255, 255, 0.07);
  }
  .icon svg {
    width: 14px;
    height: 14px;
    fill: var(--text);
    stroke: var(--text);
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .icon.main svg {
    width: 16px;
    height: 16px;
  }
  .title {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    background: none;
    border: none;
    color: var(--muted);
    font-size: 12px;
    padding: 0 6px;
  }
  .title:hover {
    color: var(--text);
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 180px;
  }
  .eq {
    display: inline-flex;
    gap: 2px;
    align-items: flex-end;
    height: 12px;
  }
  .eq i {
    width: 3px;
    background: var(--claude-soft);
    border-radius: 1px;
    animation: eq 1.1s ease-in-out infinite;
  }
  .eq i:nth-child(2) {
    animation-delay: 0.25s;
  }
  .eq i:nth-child(3) {
    animation-delay: 0.5s;
  }
  @keyframes eq {
    0%,
    100% {
      height: 3px;
    }
    50% {
      height: 12px;
    }
  }
  .panel {
    position: absolute;
    bottom: calc(100% + 8px);
    right: 0;
    z-index: 30;
    width: 340px;
    background: var(--panel-strong);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    gap: 10px;
    font-size: 13px;
  }
  :global(.home-player) .panel {
    bottom: auto;
    top: calc(100% + 8px);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .row input[type="range"] {
    flex: 1;
    accent-color: var(--claude);
  }
  .check input {
    accent-color: var(--claude);
  }
  .err {
    margin: 0;
    color: #ffb4ad;
    font-size: 12px;
    line-height: 1.5;
  }
  .hint {
    margin: 0 0 6px;
    color: var(--muted);
    font-size: 12px;
    line-height: 1.5;
  }
  code {
    display: block;
    font: 11.5px var(--mono);
    background: rgba(0, 0, 0, 0.3);
    border-radius: 6px;
    padding: 6px 8px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  .actions {
    display: flex;
    gap: 6px;
    margin-top: 8px;
  }
  .actions .btn {
    padding: 5px 10px;
    font-size: 12px;
  }
</style>
