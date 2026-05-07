# Guía de herramientas

G-MAN tiene 6 herramientas que usa para leer, modificar y validar tus dotfiles.

---

## READ — Leer archivos

Lee el contenido de un archivo. Límite: 10 KB por lectura.

```
READ: ~/.config/hypr/hyprland.conf
```

G-MAN responde mostrando el contenido. Si el archivo no existe o está fuera de los directorios permitidos, te avisa.

---

## WRITE — Escribir archivos

Escribe contenido nuevo en un archivo. **Siempre crea un backup** (`.bak`) antes de escribir.

```
WRITE: ~/.config/hypr/hyprland.conf
monitor=DP-1,1920x1080@144,0x0,1
monitor=HDMI-A-1,1920x1080@60,1920x0,1

exec-once=waybar
exec-once=hyprpaper
END
```

El contenido va entre `WRITE:` y `END`. G-MAN te muestra el diff de lo que cambió.

---

## LIST — Listar directorios

Muestra el contenido de una carpeta (nombres, sin recursión).

```
LIST: ~/.config/hypr
```

---

## RUN — Ejecutar comandos

Ejecuta comandos dentro del sandbox. Solo comandos seguros permitidos.

**Permitidos**: `grep`, `ls`, `cat`, `pacman -Q`, `systemctl --user`, `hyprctl`, `waybar`, `find`, `which`, `echo`

**Bloqueados**: `rm`, `dd`, `mkfs`, `sudo`, `su`, `chmod`, `chown`, `mount`, `reboot`, `shutdown`

```
RUN: hyprctl monitors
RUN: pacman -Qs waybar
```

---

## CHECK — Validar sintaxis

Revisa la sintaxis de archivos de configuración. Soporta tres tipos:

| Tipo | Qué valida |
|------|------------|
| `hyprland` | Llaves `{}` balanceadas, keys conocidas |
| `waybar` | JSON válido |
| `bash` | `bash -n` (errores de sintaxis) |

```
CHECK: hyprland
monitor=DP-1,1920x1080@144,0x0,1
exec-once=waybar
END
```

---

## SEARCH — Buscar en wiki

Busca en archivos `.md` dentro de `~/.config/gman/knowledge/`.

```
SEARCH: waybar config
```

Si no hay base de conocimiento, G-MAN te lo dice. Creá tus propios `.md` en esa carpeta para darle contexto.
