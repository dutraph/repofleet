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

- **Varre** seus diretórios e lista todos os repositórios git encontrados
- Mostra ao lado o **ícone do provedor** (estilo oh-my-zsh):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- Detecta **repositórios duplicados** (mesmo remoto clonado em mais de um path) e marca com `⧉ i/n`
- **Ações git em massa**: selecione vários repos e dê `pull --ff-only` ou `fetch --all --prune` de uma vez
- **Conecta ao seu servidor git via PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lista os repositórios remotos e permite **escolher qual clonar e o path local**
- Antes de clonar, **avisa se aquele repositório já está clonado** em outro path da sua máquina

### Uso

```bash
fleet                 # abre a TUI
fleet scan            # lista os repos no terminal (headless)
fleet login           # conecta um servidor git (GitHub/GitLab/Azure/Bitbucket) via PAT
fleet accounts        # lista as contas configuradas (★ = ativa)
fleet switch <nome>   # troca a conta ativa
```

Atalhos na TUI: `espaço` seleciona · `p` pull · `f` fetch · `enter` detalhes · `c` clonar do servidor · `/` filtrar · `r` rescan · `?` ajuda · `q` sair.

Os diretórios varridos ficam em `scan_roots` no config (`~/.config/repos/config.yaml`); por padrão é o seu `$HOME`.

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

- **Scans** your directories and lists every git repo found
- Shows the **provider icon** next to each (oh-my-zsh style):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- Detects **duplicate repos** (same remote cloned into more than one path) and tags them `⧉ i/n`
- **Bulk git actions**: multi-select repos and `pull --ff-only` or `fetch --all --prune` them at once
- **Connects to your git server via PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lists remote repos and lets you **pick which one to clone and the local path**
- Before cloning, **warns if that repo is already cloned** elsewhere on your machine

### Usage

```bash
fleet                 # launch the TUI
fleet scan            # list repos in the terminal (headless)
fleet login           # connect a git server (GitHub/GitLab/Azure/Bitbucket) via PAT
fleet accounts        # list configured accounts (★ = active)
fleet switch <name>   # switch the active account
```

TUI keys: `space` select · `p` pull · `f` fetch · `enter` details · `c` clone from server · `/` filter · `r` rescan · `?` help · `q` quit.

Scanned directories live under `scan_roots` in the config (`~/.config/repos/config.yaml`); defaults to your `$HOME`.

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

- **Escanea** tus directorios y lista todos los repositorios git encontrados
- Muestra el **icono del proveedor** (estilo oh-my-zsh):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local
- Detecta **repositorios duplicados** (mismo remoto clonado en más de un path) y los marca con `⧉ i/n`
- **Acciones git en lote**: selecciona varios repos y haz `pull --ff-only` o `fetch --all --prune` a la vez
- **Se conecta a tu servidor git vía PAT** (GitHub, GitLab, Azure DevOps, Bitbucket), lista los repos remotos y permite **elegir cuál clonar y el path local**
- Antes de clonar, **avisa si ese repositorio ya está clonado** en otro path de tu máquina

### Uso

```bash
fleet                 # abre la TUI
fleet scan            # lista los repos en la terminal (headless)
fleet login           # conecta un servidor git (GitHub/GitLab/Azure/Bitbucket) vía PAT
fleet accounts        # lista las cuentas configuradas (★ = activa)
fleet switch <nombre> # cambia la cuenta activa
```

Atajos en la TUI: `espacio` selecciona · `p` pull · `f` fetch · `enter` detalles · `c` clonar del servidor · `/` filtrar · `r` rescan · `?` ayuda · `q` salir.

Los directorios escaneados están en `scan_roots` del config (`~/.config/repos/config.yaml`); por defecto tu `$HOME`.

<p align="right"><a href="#repofleet">▲ volver arriba</a></p>
