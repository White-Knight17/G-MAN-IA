# G-MAN — Tu asistente Linux que no te deja tirado

Un asistente de IA 100% local que te ayuda a configurar Arch Linux + Hyprland sin tocar el kernel. Le pedís, él hace. Simple.

> **Estado**: Alpha funcional — probado con llama3.2:3b en CPU. 85 tests, 0 fallos.

---

## Quick path

```bash
# 1. Instalá Ollama y bajá el modelo
ollama pull llama3.2:3b

# 2. Cloná y ejecutá
git clone https://github.com/gentleman/gman.git
cd gman
go run ./cmd/harvey
```

Listo. Se abre la TUI. Escribí lo que necesités.

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

**Seguridad**: Sandbox con Bubblewrap. Path traversal bloqueado. Comandos peligrosos (`rm`, `sudo`, `dd`) prohibidos. Solo toca lo que vos autorizás.

---

## Requisitos

| Qué | Mínimo |
|-----|--------|
| RAM | 8 GB |
| CPU | 4 cores |
| Disco | 3 GB (modelo + binario) |
| Ollama | v0.23+ |
| Go | 1.26+ |
| SO | Linux (Arch, CachyOS probado) |

El binario compilado pesa ~15 MB. Sin Python, sin Node, sin dependencias externas.

---

## Detalles técnicos

| Tema | Decisión |
|------|----------|
| Lenguaje | Go 1.26 — binario único, sin runtime |
| Arquitectura | Clean / Hexagonal — dominio puro, infraestructura intercambiable |
| Modelo | llama3.2:3b vía Ollama (local, sin internet) |
| Sandbox | Bubblewrap + path validation defense-in-depth |
| TUI | Bubbletea + Lipgloss — chat con split view y preview de archivos |
| Tests | 85 tests, 7 paquetes, 0 fallos |

---

## Opciones

```bash
go run ./cmd/harvey \
  --model llama3.2:3b \
  --ollama-url http://localhost:11434 \
  --allowed-dirs ~/.config,~/.local
```

---

## Base de conocimiento

G-MAN busca en archivos `.md` dentro de `~/.config/harvey/knowledge/`. Creá los tuyos:

```
~/.config/harvey/knowledge/
├── hyprland.md      ← Cómo configurar monitores, workspaces, atajos
├── waybar.md        ← Barras, widgets, estilos
├── arch-pacman.md   ← Pacman, AUR, tips
└── lo-que-quieras.md
```

---

## Roadmap

- [x] Agente ReAct con 6 herramientas sandboxeadas
- [x] TUI con chat, preview de archivos, diálogos de permisos
- [x] CI/CD con GitHub Actions
- [x] Release Please (versionado automático)
- [ ] Streaming de respuestas
- [ ] Soporte para APIs remotas (OpenAI, Anthropic)
- [ ] Persistencia de sesiones e historial
- [ ] Plugins y herramientas custom

---

## Contribuir

```bash
git clone https://github.com/gentleman/gman.git
cd gman
go test ./...           # 85 tests deben pasar
go vet ./...            # sin warnings
go build ./cmd/harvey   # binario en ./harvey
```

Commits en [Conventional Commits](https://www.conventionalcommits.org/). PRs requieren tests.

---

Hecho con ❤️ para que Linux sea menos dolor de cabeza.
