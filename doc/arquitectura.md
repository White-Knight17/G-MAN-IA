# Arquitectura

G-MAN sigue Clean Architecture con cuatro capas. Las dependencias apuntan hacia adentro — las capas externas nunca son importadas por las internas.

```
cmd/harvey/main.go          ← Composición root (DI)
    │
    ├── internal/domain/     ← Entidades, value objects, interfaces
    │   ├── agent.go         │   Agent interface (ReAct loop)
    │   ├── tool.go          │   Tool interface + ToolResult
    │   ├── sandbox.go       │   Sandbox interface
    │   ├── permission.go    │   PermissionRepository, Grant
    │   └── session.go       │   Session, ChatMessage
    │
    ├── internal/application/ ← Casos de uso, orquestación
    │   ├── orchestrator.go  │   ChatOrchestrator (ReAct loop)
    │   ├── executor.go      │   ToolExecutor (parser + router)
    │   └── grantmgr.go      │   GrantManager
    │
    ├── internal/infrastructure/ ← Adaptadores concretos
    │   ├── ollama/          │   OllamaClient (HTTP → /api/chat)
    │   ├── permission/      │   InMemoryPermissionRepo
    │   ├── sandbox/         │   BubblewrapSandbox + LandlockSandbox
    │   └── tools/           │   6 tool implementations
    │
    └── internal/ui/tui/     ← Interfaz de usuario
        ├── model.go         │   Bubbletea Model
        ├── update.go        │   Handlers de eventos
        ├── view.go          │   Renderizado
        └── styles.go        │   Tema Lipgloss
```

## Principios

| Principio | Aplicación |
|-----------|------------|
| **Domain puro** | Cero imports externos. Solo tipos de Go. |
| **Puertos e interfaces** | Domain define QUÉ. Infrastructure define CÓMO. |
| **Inversión de dependencias** | UI → Application ← Infrastructure. Nunca al revés. |
| **Testeable** | Cada capa se testea aislada con mocks. 85 tests. |

---

## Flujo de una conversación

```
1. Usuario escribe "ordená mi hyprland.conf"
2. TUI → ChatOrchestrator.HandleMessage()
3. Orchestrator → Agent.Run() → OllamaClient → POST /api/chat
4. Ollama responde: "READ: ~/.config/hypr/hyprland.conf"
5. Orchestrator → ToolExecutor.Execute()
6. Executor parsea "READ:" → busca Tool → verifica permisos → ejecuta
7. Tool (ReadFileTool) lee archivo vía Sandbox → ToolResult
8. Orchestrator → Agent.Run() con resultado → Ollama genera respuesta final
9. Orchestrator → TUI: muestra respuesta y preview del archivo
```

---

## Seguridad

G-MAN usa defensa en profundidad:

| Capa | Qué protege |
|------|-------------|
| **Path validation** | Resuelve y normaliza paths. Bloquea `../` traversal. |
| **Bubblewrap** | Aísla comandos en contenedor sin acceso al sistema. |
| **Landlock** (pendiente) | Restricción in-process a nivel kernel. |
| **Command allowlist** | Solo comandos seguros (`grep`, `ls`, `cat`, etc.). |
| **Permission model** | El usuario autoriza cada carpeta antes de tocarla. |
