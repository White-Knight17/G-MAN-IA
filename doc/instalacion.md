# Guía de instalación

## 1. Instalar Ollama

```bash
# Arch / CachyOS
sudo pacman -S ollama

# Iniciar el servicio
systemctl --user enable --now ollama
```

## 2. Bajar el modelo

```bash
ollama pull llama3.2:3b
```

Verificá que esté disponible:

```bash
ollama list
# NAME            ID              SIZE      MODIFIED
# llama3.2:3b     a80c4f17acd5    2.0 GB    ...
```

## 3. Compilar G-MAN

```bash
git clone https://github.com/gentleman/gman.git
cd gman
go build -o gman ./cmd/harvey
```

## 4. Ejecutar

```bash
./gman
```

### Con opciones personalizadas

```bash
./gman --model qwen2.5:3b --allowed-dirs ~/.config,~/proyectos
```

## 5. Verificar que funciona

```bash
bash scripts/e2e-test.sh
```

Salida esperada:

```
Total:  5
Passed: 5 ✅
Failed: 0
```
