# G-MAN — Tu asistente Linux que no te deja tirado

> **Built with Tauri + Svelte 5 + Go** — Native desktop GUI, private AI, 100% local.

Un asistente de IA 100% local que te ayuda a configurar Arch Linux + Hyprland sin tocar el kernel. Le pedís, él hace. Simple.

> **Estado**: v1.0.0 — Desktop GUI funcional. 330 tests, 0 fallos.

---

## Quick path

```bash
# 1. Instalá Ollama y bajá el modelo
ollama pull llama3.2:3b

# 2. Cloná, instalá dependencias y ejecutá
git clone https://github.com/gentleman/gman.git
cd gman
make install
make dev
```

Listo. Se abre la ventana de G-MAN. Escribí lo que necesités.

---

## Screenshots

<!-- TODO: Add screenshots -->
| Chat | Onboarding | Permissions |
|------|------------|-------------|
| ![](./doc/screenshots/chat.png) | ![](./doc/screenshots/wizard.png) | ![](./doc/screenshots/permissions.png) |

---

## Qué hace

| Herramienta | Comando | Qué hace |
|-------------|---------|----------|
| Leer archivos | `READ: /path` | Muestra contenido de dotfiles |
| Escribir archivos | `WRITE: /path` | Modifica archivos con backup `.bak` |
| Listar directorios | `LIST: /path` | Muestra qué hay en una carpeta |
| Ejecutar comandos | `RUN: comando` | Solo comandos seguros (allowlist) |
| Chequear sintaxis | `CHECK: tipo` | Valida configs de Hyprland, Waybar, Bash |
| Buscar en wiki | `SEARCH: query` | Busca en tu base de conocimiento local |

**Interfaz GUI**: Chat con burbujas, streaming de tokens en tiempo real, diálogos de permisos modales, wizard de onboarding, system tray con Show/Hide/Quit, atajo de teclado `Ctrl+Shift+G` para toggle del chat.

**Seguridad**: Sandbox con Bubblewrap. Path traversal bloqueado. Comandos peligrosos (`rm`, `sudo`, `dd`) prohibidos. Solo toca lo que vos autorizás.

---

## Requisitos

| Qué | Mínimo |
|-----|--------|
| RAM | 8 GB |
| CPU | 4 cores |
| Disco | 3 GB (modelo + binario) |
| Ollama | v0.23+ |
| Node.js | 22+ |
| pnpm | 9+ |
| Rust | 1.80+ |
| Go | 1.26+ |
| SO | Linux (Arch, CachyOS probado) |

---

## Build

```bash
# Build completo (Go sidecar + Svelte frontend + Tauri bundle)
make build

# Solo el sidecar Go
make build-core

# Solo el frontend Svelte
make build-ui

# Bundles para distribución (.deb, .AppImage, .rpm)
make bundle

# Ejecutar todos los tests
make test-all

# Modo desarrollo
make dev

# Limpiar artefactos
make clean
```

---

## Detalles técnicos

| Tema | Decisión |
|------|----------|
| Lenguaje core | Go 1.26 — binario único, sin runtime |
| Desktop shell | Tauri v2 + Rust — nativo, liviano |
| Frontend | Svelte 5 (runes) + TypeScript |
| Arquitectura | Clean / Hexagonal — dominio puro, infraestructura intercambiable |
| Modelo | llama3.2:3b vía Ollama (local, sin internet) |
| Transporte | JSON-RPC 2.0 NDJSON sobre stdin/stdout del sidecar |
| Sandbox | Bubblewrap + path validation defense-in-depth |
| Tests | 330 tests (Go: 264, Svelte: 57, E2E: 9) |
| CI/CD | GitHub Actions con test-go + test-rust + test-frontend + build |

---

## Base de conocimiento

G-MAN busca en archivos `.md` dentro de `~/.config/gman/knowledge/`. Creá los tuyos:

```
~/.config/gman/knowledge/
├── hyprland.md      ← Cómo configurar monitores, workspaces, atajos
├── waybar.md        ← Barras, widgets, estilos
├── arch-pacman.md   ← Pacman, AUR, tips
└── lo-que-quieras.md
```

---

## Roadmap

- [x] Agente ReAct con 6 herramientas sandboxeadas
- [x] Desktop GUI con Tauri v2 + Svelte 5 (streaming, permisos, onboarding)
- [x] Streaming de respuestas
- [x] CI/CD con GitHub Actions (Go + Rust + Svelte + build)
- [x] Empaquetado para Linux (.deb, .AppImage, .rpm)
- [ ] Soporte para APIs remotas (OpenAI, Anthropic)
- [ ] Persistencia de sesiones e historial
- [ ] Plugins y herramientas custom

---

## Contribuir

```bash
git clone https://github.com/gentleman/gman.git
cd gman
make test-all          # Todos los tests deben pasar
cd core && go vet ./... # sin warnings
make build             # build completo
```

Commits en [Conventional Commits](https://www.conventionalcommits.org/). PRs requieren tests.

---

Hecho con ❤️ para que Linux sea menos dolor de cabeza.
