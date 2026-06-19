# repofleet

> a TUI to manage all your local and remote git repositories · gerenciador TUI para todos os seus repositórios git locais e remotos · un gestor TUI para todos tus repositorios git locales y remotos

<p align="center">
  <strong><a href="#português">Português</a></strong>
  ·
  <strong><a href="#english">English</a></strong>
  ·
  <strong><a href="#español">Español</a></strong>
</p>

---

## Português

gerenciador TUI para todos os seus repositórios git locais e remotos

### Instalação

#### Opção A — `go install` (mais rápido)

```bash
go install github.com/dutraph/repofleet/cmd/fleet@latest
```

Garanta que `$(go env GOBIN)` (ou `$(go env GOPATH)/bin`) está no `PATH`. O nome do binário fica `fleet`.

#### Opção B — script de instalação

```bash
curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
# ou com prefix custom:
PREFIX="$HOME/.local" curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
# ou uma tag específica:
REF=v1.0.0 curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
```

#### Opção C — release binário

Baixe o binário pré-compilado da [página de releases](https://github.com/dutraph/repofleet/releases) (darwin / linux × amd64 / arm64), confira o SHA256 em `SHA256SUMS` e copie para um diretório do PATH.

#### Opção D — build local

```bash
git clone https://github.com/dutraph/repofleet.git
cd repofleet
make                      # build local
make install              # auto-bump do patch + instala em /usr/local/bin
make install VERSION=1.1.0
make install PREFIX=$HOME/.local
make version
make uninstall
```

`make help` lista todos os targets.

### Pré-requisitos

- Go 1.22+
- `git` no PATH
- uma [Nerd Font](https://www.nerdfonts.com) no terminal (para os ícones de provedor)

### O que faz

- **Varre** seus diretórios e lista todos os repositórios git, com o **ícone do provedor** ao lado (estilo oh-my-zsh):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- **Busca** (`/`) e **filtro por tipo** (`t`) — escolha um provedor e veja só os repos dele
- Detecta **duplicados** (mesmo remoto em mais de um path), marca com `⧉ i/n` e tem uma **tela dedicada** (`D`) que agrupa por repo e lista os paths de cada cópia
- **Ações git em massa**: `pull --ff-only`, `pull --prune`, `fetch`, e **fetch all** (sincroniza todos de uma vez)
- **Troca de branch** (`b`) e **remoção** da cópia local (`d`, com confirmação)
- **Barra de comando `:`** para rodar qualquer comando git no repo selecionado, de qualquer tela — inclusive interativos (`commit`, `rebase -i`, `add -p`)
- **Conecta ao seu servidor git via PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lista os repos remotos, deixa escolher **HTTPS ou SSH** e **navega o filesystem** (com filtro e **criar pasta**) para definir onde clonar
- Antes de clonar, **avisa se aquele repositório já está clonado** em outro path da sua máquina

### Uso

```bash
fleet                 # abre a TUI
fleet scan            # lista os repos no terminal (headless)
fleet login           # conecta um servidor git (GitHub/GitLab/Azure/Bitbucket) via PAT
fleet accounts        # lista as contas configuradas (★ = ativa)
fleet switch <nome>   # troca a conta ativa
```

Atalhos na TUI: `espaço` selecionar · `a` todos · `p` pull · `P` pull --prune · `f` fetch · `F` fetch all (sync) · `b` trocar branch · `d` remover · `D` duplicados · `t` filtrar por tipo · `/` buscar · `:` comando git · `c` clonar do servidor · `enter` detalhes · `r` rescan · `?` ajuda · `q` sair.

Os diretórios varridos ficam em `scan_roots` no config (`~/.config/fleet/config.yaml`); por padrão é o seu `$HOME`.

<p align="right"><a href="#repofleet">▲ back to top</a></p>

---

## English

a TUI to manage all your local and remote git repositories

### Installation

#### Option A — `go install` (fastest)

```bash
go install github.com/dutraph/repofleet/cmd/fleet@latest
```

#### Option B — install script

```bash
curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
PREFIX="$HOME/.local" curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
REF=v1.0.0 curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
```

#### Option C — pre-built binary

Grab from [releases](https://github.com/dutraph/repofleet/releases), verify SHA256, drop onto PATH.

#### Option D — local build

```bash
git clone https://github.com/dutraph/repofleet.git
cd repofleet
make
make install
make install VERSION=1.1.0
make install PREFIX=$HOME/.local
make version
make uninstall
```

### Requirements

- Go 1.22+
- `git` on PATH
- a [Nerd Font](https://www.nerdfonts.com) in your terminal (for the provider icons)

### What it does

- **Scans** your directories and lists every git repo, with the **provider icon** next to each (oh-my-zsh style):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- **Search** (`/`) and **filter by type** (`t`) — pick a provider and see only its repos
- Detects **duplicates** (same remote in more than one path), tags them `⧉ i/n`, and has a **dedicated view** (`D`) that groups by repo and lists each copy's path
- **Bulk git actions**: `pull --ff-only`, `pull --prune`, `fetch`, and **fetch all** (sync every repo at once)
- **Switch branch** (`b`) and **remove** the local copy (`d`, with confirmation)
- **`:` command bar** to run any git command on the selected repo, from any screen — including interactive ones (`commit`, `rebase -i`, `add -p`)
- **Connects to your git server via PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lists remote repos, lets you choose **HTTPS or SSH** and **browse the filesystem** (with filter and **create-folder**) to pick where to clone
- Before cloning, **warns if that repo is already cloned** elsewhere on your machine

### Usage

```bash
fleet                 # launch the TUI
fleet scan            # list repos in the terminal (headless)
fleet login           # connect a git server (GitHub/GitLab/Azure/Bitbucket) via PAT
fleet accounts        # list configured accounts (★ = active)
fleet switch <name>   # switch the active account
```

TUI keys: `space` select · `a` all · `p` pull · `P` pull --prune · `f` fetch · `F` fetch all (sync) · `b` switch branch · `d` remove · `D` duplicates · `t` filter by type · `/` search · `:` git command · `c` clone from server · `enter` details · `r` rescan · `?` help · `q` quit.

Scanned directories live under `scan_roots` in the config (`~/.config/fleet/config.yaml`); defaults to your `$HOME`.

<p align="right"><a href="#repofleet">▲ back to top</a></p>

---

## Español

un gestor TUI para todos tus repositorios git locales y remotos

### Instalación

#### Opción A — `go install` (la más rápida)

```bash
go install github.com/dutraph/repofleet/cmd/fleet@latest
```

#### Opción B — script de instalación

```bash
curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
PREFIX="$HOME/.local" curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
REF=v1.0.0 curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
```

#### Opción C — binario pre-compilado

Descarga desde [releases](https://github.com/dutraph/repofleet/releases), verifica el SHA256, copia al PATH.

#### Opción D — build local

```bash
git clone https://github.com/dutraph/repofleet.git
cd repofleet
make
make install
make install VERSION=1.1.0
make install PREFIX=$HOME/.local
make version
make uninstall
```

### Requisitos

- Go 1.22+
- `git` en el PATH
- una [Nerd Font](https://www.nerdfonts.com) en la terminal (para los iconos de proveedor)

### Qué hace

- **Escanea** tus directorios y lista todos los repositorios git, con el **icono del proveedor** al lado (estilo oh-my-zsh):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- **Búsqueda** (`/`) y **filtro por tipo** (`t`) — elige un proveedor y ve solo sus repos
- Detecta **duplicados** (mismo remoto en más de un path), los marca con `⧉ i/n` y tiene una **pantalla dedicada** (`D`) que agrupa por repo y lista el path de cada copia
- **Acciones git en lote**: `pull --ff-only`, `pull --prune`, `fetch`, y **fetch all** (sincroniza todos a la vez)
- **Cambiar de branch** (`b`) y **eliminar** la copia local (`d`, con confirmación)
- **Barra de comandos `:`** para ejecutar cualquier comando git en el repo seleccionado, desde cualquier pantalla — incluso interactivos (`commit`, `rebase -i`, `add -p`)
- **Se conecta a tu servidor git vía PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lista los repos remotos, permite elegir **HTTPS o SSH** y **navegar el filesystem** (con filtro y **crear carpeta**) para definir dónde clonar
- Antes de clonar, **avisa si ese repositorio ya está clonado** en otro path de tu máquina

### Uso

```bash
fleet                 # abre la TUI
fleet scan            # lista los repos en la terminal (headless)
fleet login           # conecta un servidor git (GitHub/GitLab/Azure/Bitbucket) vía PAT
fleet accounts        # lista las cuentas configuradas (★ = activa)
fleet switch <nombre> # cambia la cuenta activa
```

Atajos en la TUI: `espacio` seleccionar · `a` todos · `p` pull · `P` pull --prune · `f` fetch · `F` fetch all (sync) · `b` cambiar branch · `d` eliminar · `D` duplicados · `t` filtrar por tipo · `/` buscar · `:` comando git · `c` clonar del servidor · `enter` detalles · `r` rescan · `?` ayuda · `q` salir.

Los directorios escaneados están en `scan_roots` del config (`~/.config/fleet/config.yaml`); por defecto tu `$HOME`.

<p align="right"><a href="#repofleet">▲ volver arriba</a></p>
