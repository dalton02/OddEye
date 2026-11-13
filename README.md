# Oddeye

App desktop (Linux, Windows e macOS) que fica na bandeja do sistema vigiando os repositórios que você
escolher. Quando chega uma pull request nova, ele avisa com uma notificação. Um clique abre o tour da PR
com o Claude Code: o Claude escolhe só os arquivos críticos (sem testes nem documentação) e explica cada
um. O código aparece no editor do VS Code (Monaco) e os seus comentários vão para o GitHub como review.

Nada é publicado sem você: o review só sai quando você clica em **Enviar review** e escolhe Comentar,
Aprovar ou Pedir mudanças.

## Instalação

Baixe o instalador do seu sistema na página de [Releases](../../releases):

| Sistema | Arquivo | Observação |
| ------- | ------- | ---------- |
| Windows 10/11 | `Oddeye-<versão>-windows-setup.exe` | Também há um `.exe` portátil. O WebView2 já vem no Windows. |
| macOS (Intel e Apple Silicon) | `Oddeye-<versão>-macos-universal.dmg` | Veja abaixo como abrir um app sem assinatura da Apple. |
| Debian, Ubuntu, Mint | `oddeye_<versão>_amd64.deb` | `sudo apt install ./oddeye_<versão>_amd64.deb` |
| Fedora, openSUSE | `oddeye-<versão>.x86_64.rpm` | `sudo dnf install ./oddeye-<versão>.x86_64.rpm` |
| Outras distros | `Oddeye-<versão>-x86_64.AppImage` | Precisa do `libwebkit2gtk-4.1` instalado no sistema. |

**macOS:** o app ainda não tem assinatura da Apple, então o Gatekeeper bloqueia a primeira abertura.
Arraste o app para Aplicativos e rode uma vez no terminal:

```bash
xattr -dr com.apple.quarantine "/Applications/Oddeye.app"
```

**Linux com GNOME:** o GNOME não mostra ícones de bandeja sem a extensão
[AppIndicator](https://extensions.gnome.org/extension/615/appindicator-support/) (Ubuntu já traz ela
ligada). KDE, Cinnamon, XFCE e afins funcionam direto.

## Pré-requisitos

O Oddeye não guarda token nenhum: ele usa as ferramentas que você já tem logadas.

- [GitHub CLI](https://cli.github.com) (`gh`). Se não estiver logado, o próprio app faz o login: ele mostra
  um código, abre a página do GitHub e você só confirma. O app pede os escopos `repo` (ler PRs, inclusive
  privadas, e publicar reviews) e `read:org` (listar os repositórios das suas organizações).
- [Claude Code](https://claude.com/claude-code) (`claude`) instalado e logado. O revisor roda no seu
  Claude Code em modo headless (`claude -p`), usando o seu plano: não precisa de chave de API. Se o
  Claude ainda não tiver login, abra `claude` no terminal uma vez e faça `/login`.

## Como usar

1. Na primeira abertura, o app confere o `gh` e o Claude Code e guia o que faltar.
2. Clique em **Escolher repositórios** e adicione os repositórios que quer vigiar (busca entre os seus,
   ou digite `dono/repo`).
3. Feche a janela: o app continua na bandeja. Ele já vem configurado para abrir junto com o sistema, direto na bandeja (dá para desligar no aviso da primeira tela ou em **Configurações**). A cada 5 minutos (configurável) ele procura PRs novas.
   O ícone ganha um ponto vermelho quando tem PR esperando, e o menu da bandeja lista as PRs.
4. Clique na notificação, numa PR do menu ou na caixa de entrada da tela inicial para abrir o tour.
   Também dá para colar o link de qualquer PR (`https://github.com/org/repo/pull/123` ou `org/repo#123`).
5. Use **← →** para navegar entre os arquivos. Clique no número da linha (ou selecione linhas e aperte
   **C**) para comentar. Os comentários ficam como rascunho em disco até você clicar em **Enviar review**.

Na primeira varredura de um repositório recém-adicionado, as PRs que já estavam abertas são marcadas como
vistas sem notificar, para não chegar uma enxurrada de avisos de uma vez. Para sair do app de verdade,
use **Sair** no menu da bandeja.

### Opções por repositório

- **Só avisar quando pedirem o meu review**: ignora PRs em que você não é revisor.
- **Pré-analisar com o Claude ao chegar**: roda a triagem em segundo plano assim que a PR chega, e abrir
  o tour fica instantâneo. Cada PR nova gasta uma análise do seu plano do Claude, por isso vem desligado.
- **Instruções próprias**: um prompt de revisor só para aquele repositório.

### Prompt do revisor

Em **Configurações → Revisor (Claude)** você edita as instruções do revisor: o que priorizar, o tom e o
idioma. As regras de formato da resposta (caminhos de arquivo, números de linha, JSON) são do app e
entram sempre, então um prompt customizado não quebra o tour. Trocar o prompt ou o modelo gera uma
análise nova; sem troca, a análise fica em cache por commit da PR e reabrir não gasta outra.

O modelo também pode ser escolhido ali (vazio usa o padrão do seu Claude Code). A variável de ambiente
`ODDEYE_MODEL` tem prioridade sobre a configuração.

## Música

O app não traz música embutida. Coloque suas faixas (mp3, ogg, flac, wav, m4a) na pasta
`~/.config/oddeye/music` (ou escolha outra no controle de som). Elas tocam em ordem alfabética, em
loop. Sem faixas na pasta, toca uma trilha ambiente sintetizada pelo próprio app.

## Dados locais

| O quê | Linux | macOS | Windows |
| ----- | ----- | ----- | ------- |
| Configuração (`config.json`) e estado do vigia (`state.json`) | `~/.config/oddeye/` | `~/Library/Application Support/oddeye/` | `%AppData%\oddeye\` |
| Rascunhos de comentário | `.../oddeye/drafts/` | idem | idem |
| Cache das análises | `~/.cache/oddeye/analysis/` | `~/Library/Caches/oddeye/analysis/` | `%LocalAppData%\oddeye\analysis\` |

O diff da PR vai para o Claude pelo seu Claude Code; nada passa por servidor do Oddeye (não existe um).

## Desenvolvimento

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
wails dev -tags webkit2_41          # Linux com WebKitGTK 4.1
go test ./internal/...
cd frontend && npm run check && npm test
```

Estrutura:

- `internal/gh`: tudo que fala com o GitHub, via `gh` (PRs, arquivos, reviews, busca das PRs abertas, login).
- `internal/claude`: triagem com o `claude -p`, prompt (parte editável + contrato de formato fixo).
- `internal/watcher`: varredura periódica e a regra do que conta como PR nova (`Evaluate`).
- `internal/tray`: ícone e menu da bandeja (fyne.io/systray no Linux e Windows, `NSStatusItem` no macOS).
- `internal/config`, `internal/autostart`, `internal/review`, `internal/music`.

## Build e release

```bash
wails build -tags webkit2_41                 # Linux → build/bin/oddeye
wails build -platform windows/amd64          # Windows (cross-compile do Linux) → build/bin/oddeye.exe
```

O macOS precisa ser compilado num Mac (a bandeja usa cgo). O workflow `.github/workflows/build.yml` roda
os testes em todo push e PR, e numa tag `v*` gera os instaladores dos três sistemas e publica o release
com os `SHA256SUMS`.

## Licença

[MIT](LICENSE)
