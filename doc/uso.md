# Guía de uso

## La TUI

Cuando ejecutás `./gman`, se abre una interfaz de terminal con tres zonas:

```
┌──────────────────────────────────────────────────────────┐
│ G-MAN — llama3.2:3b                          Ctrl+C salir │
├────────────────────────────────┬─────────────────────────┤
│                                │                         │
│  [Harvey] ¡Hola! ¿En qué te   │   ~/.config/hypr/       │
│  ayudo hoy?                   │   hyprland.conf         │
│                                │                         │
│  [You] Ordename el             │   monitor=DP-1,...      │
│  hyprland.conf                │   exec-once=waybar      │
│                                │                         │
│  [Harvey] Dale, voy a leerlo  │                         │
│  y te muestro qué cambios     │                         │
│  hacer...                     │                         │
│                                │                         │
├────────────────────────────────┴─────────────────────────┤
│ > Ordename el hyprland.conf                           ▌ │
└──────────────────────────────────────────────────────────┘
```

- **Izquierda (70%)**: Chat. Ves tus mensajes (`[You]`) y las respuestas (`[Harvey]`).
- **Derecha (30%)**: Preview. Muestra el contenido del último archivo que G-MAN leyó o escribió.
- **Abajo**: Barra de input. Escribí tu pregunta y apretá Enter.

---

## Atajos de teclado

| Tecla | Acción |
|-------|--------|
| `Enter` | Enviar mensaje |
| `Ctrl+C` | Salir |
| `Esc` | Cerrar diálogo de permisos / Salir |
| `q` | Salir |
| `↑` / `↓` | Scroll en el chat |
| `Tab` | Alternar preview |

---

## Diálogo de permisos

La primera vez que G-MAN necesita tocar una carpeta, te pide permiso:

```
┌──────────────────────────────────────┐
│                                      │
│   ¿Permitir a G-MAN escribir en     │
│   ~/.config/hypr/?                  │
│                                      │
│        [Y] Sí    [N] No             │
│                                      │
└──────────────────────────────────────┘
```

Apretá `Y` para autorizar o `N` para rechazar. Los permisos duran hasta que cerrás G-MAN.

---

## Ejemplos de conversación

### Configurar monitores

```
You: Tengo dos monitores, DP-1 y HDMI-A-1. Configuralos en hyprland.

Harvey: Voy a leer tu config actual primero.
> READ: ~/.config/hypr/hyprland.conf

[Preview muestra el archivo actual]

Harvey: Veo que tenés solo un monitor. Voy a agregar el segundo.
¿Me permitís escribir en ~/.config/hypr/?

[Diálogo de permisos — apretás Y]

Harvey: Listo. Agregué ambos monitores con sus resoluciones.
El backup quedó en hyprland.conf.bak por si querés volver atrás.
```

### Buscar información

```
You: Cómo instalo waybar?

Harvey: Busco en la wiki...
> SEARCH: waybar install

[Busca en ~/.config/harvey/knowledge/*.md]

Harvey: Encontré esto en waybar.md:
waybar se instala con: sudo pacman -S waybar
Se configura en ~/.config/waybar/config.jsonc
```

### Validar configuración

```
You: Chequeame esta config de waybar

CHECK: waybar
{
    "layer": "top",
    "position": "top",
    "modules-left": ["hyprland/workspaces"],
    "modules-right": ["clock"]
}
END

Harvey: Lo revisé. La sintaxis JSON es válida.
Todo parece correcto.
```
