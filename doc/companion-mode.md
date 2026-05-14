# Companion Mode (v2.1.0)

G-MAN evoluciona de una ventana de chat flotante a un **compañero de escritorio permanente** con soporte para múltiples APIs de IA.

## Novedades

### 1. Modos de ventana

G-MAN ahora soporta tres modos de visualización:

| Modo | Comportamiento |
|------|---------------|
| **Floating** (default) | Ventana 420×700, centrada, modo tradicional |
| **Companion** | Sidebar 380px anclado al borde derecho, always-on-top, altura completa |
| **Compact** | Barra fina (48px) anclada al borde, se expande al hacer hover |

**Atajo global**: `Ctrl+Shift+G` para mostrar/ocultar G-MAN en cualquier momento.

Actualmente los modos se configuran editando `~/.config/gman/config.json`:
```json
{
  "window": {
    "mode": "companion",
    "width": 380
  }
}
```

### 2. Slash Commands

Escribí `/` en el chat para acceder a comandos rápidos. Se procesan localmente (sin consumir tokens de IA).

| Comando | Descripción |
|---------|------------|
| `/help` | Muestra todos los comandos disponibles |
| `/clear` | Limpia el historial del chat |
| `/model` | Muestra configuración actual: provider, modelo, API key |
| `/model <nombre>` | Cambia de modelo (ej: `/model gpt-4o`) |
| `/models <nombre>` | Descarga un modelo de Ollama (`/models qwen2.5:3b`) |
| `/api <provider> <key>` | Configura API key remota (ver abajo) |

Al escribir `/`, aparece automáticamente una palette de autocompletado.

### 3. APIs remotas (OpenAI, DeepSeek, Groq, etc.)

G-MAN ahora soporta cualquier API compatible con OpenAI, además del Ollama local.

#### Configurar una API remota

```bash
/api deepseek sk-xxxxxxxxxxxxxxxx
```

G-MAN auto-detecta:
- **Proveedor**: `deepseek`, `openai`, `groq`, etc.
- **Modelo por defecto**: `deepseek-v4-pro`, `gpt-4o`, `llama-3.3-70b-versatile`
- **URL base**: `api.deepseek.com`, `api.openai.com`, etc.

#### Cambiar de modelo

```bash
/model deepseek-v4-flash
```

#### Volver a Ollama

Simplemente no configures una API key. Si ya tenés una, editá `~/.config/gman/config.json` y borrala, o cambiá el provider a `ollama`.

### 4. Configuración persistente

La configuración se guarda en `~/.config/gman/config.json`:

```json
{
  "version": "2.1.0",
  "theme": "dark",
  "backend": {
    "provider": "deepseek",
    "model": "deepseek-v4-pro",
    "ollama_url": "http://localhost:11434",
    "api_keys": {
      "deepseek": "sk-xxx..."
    }
  },
  "window": {
    "mode": "floating",
    "width": 420
  }
}
```

### 5. Material UI Refresh

La UI ahora usa principios de Material Design con CSS puro:

- **Elevation**: 4 niveles de sombras (`--gman-elevation-1` a `4`)
- **Spacing**: Sistema de 8px (`--gman-space-xs` a `xl`)
- **Tipografía**: Jerarquía consistente (title, body, caption)
- **Transiciones**: Hover/active states en botones con escala

## Migración desde v2.0.0

Si tenías config en localStorage, se migra automáticamente a `config.json` en el primer inicio de v2.1.0. La flag `gman-migrated=true` en localStorage evita re-migraciones.

## Requisitos

Igual que v2.0.0. Para APIs remotas necesitás conexión a internet y una API key válida.
