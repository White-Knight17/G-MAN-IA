# Slash Commands

G-MAN soporta **comandos rápidos** que se escriben directamente en el input del chat. Empezá con `/` y G-MAN intercepta el comando en vez de enviarlo a la IA.

## Comandos disponibles

| Comando | Qué hace | Ejemplo |
|---------|----------|---------|
| `/help` | Muestra todos los comandos disponibles | `/help` |
| `/clear` | Limpia el historial del chat actual | `/clear` |
| `/model` | Muestra el modelo activo y los modelos locales disponibles | `/model` |
| `/models <nombre>` | Descarga un modelo nuevo desde Ollama con progreso en vivo | `/models qwen2.5:3b` |

## Cómo usarlos

### /help

Muestra la lista completa de comandos con descripción. Ideal si no te acordás qué hay disponible.

```
/help
```

### /clear

Borra todos los mensajes del chat actual. No afecta la configuración ni los permisos.

```
/clear
```

### /model

Muestra:
- El **modelo activo** actualmente
- El **provider** configurado (local Ollama o remoto)
- La lista de **modelos disponibles** en tu Ollama local con su tamaño

El modelo activo aparece marcado con `(active)`.

```
/model
```

Salida de ejemplo:
```
Current model: llama3.2:3b
Provider: local

Available models:
  • llama3.2:3b — 2.0 GB (active)
  • qwen2.5:3b — 1.8 GB
  • phi3:mini — 2.3 GB
```

### /models <nombre>

Descarga un modelo nuevo desde Ollama. Muestra el progreso de descarga en tiempo real.

```
/models qwen2.5:7b
```

Mientras descarga, ves notificaciones de progreso con el status y porcentaje. Cuando termina, el modelo queda disponible para usar con `/model`.

> **Nota**: Los modelos pueden ser grandes. Verificá tener espacio en disco antes de descargar.

## Command Palette

Cuando escribís `/`, aparece automáticamente una **lista de autocompletado** con los comandos disponibles. Podés:

- Seguir escribiendo para filtrar (`/mo` muestra solo `/model` y `/models`)
- Click en un comando para seleccionarlo
- Presionar Enter para ejecutar el comando completo

## Detalle técnico

Los comandos se procesan **localmente en el frontend** (Svelte). No viajan al modelo de IA, así que son instantáneos y no consumen tokens.

Algunos comandos (`/model`, `/models`) hacen llamadas JSON-RPC al sidecar Go para obtener información de Ollama.
