# 🎙️ Viz - Decentralized P2P Voice Communication via tunnels

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![WebSocket](https://img.shields.io/badge/Protocol-WebSocket-orange.svg)](https://tools.ietf.org/html/rfc6455)
[![Audio Codec](https://img.shields.io/badge/Codec-OPUS-red.svg)](https://opus-codec.org/)
[![Architecture](https://img.shields.io/badge/Architecture-P2P%20via%20Tunnels-purple.svg)]()

**Viz** is a decentralized P2P voice communication application written in Go. It solves the Symmetric NAT problem through tunneling services, allowing users to choose any intermediary servers or use their own VPS.

## ✨ Features

- 🌐 **Decentralized Architecture**: P2P communication via tunneling services
- 🚫 **NAT Bypass**: Solves Symmetric NAT problems
- 🔧 **Server Flexibility**: Any tunneling services (ngrok, cloudflare, localhost.run)
- 🎵 **High-Quality Audio**: OPUS codec with 32 kbps bitrate
- 📦 **Aggressive Compression**: OPUS + Zstandard for traffic minimization
- ⏱️ **Optimized Chunks**: 40ms audio chunks with 320ms batch delay for tunneling services
- 🔄 **Bidirectional Communication**: Simultaneous recording and playback
- 🛡️ **Thread-Safe**: Safe multi-threaded audio processing
- 📊 **Detailed Logging**: Zap integration for monitoring

## 🏗️ Architecture

```
┌──────────────────┐    Tunnel Service    ┌──────────────────┐
│   User1 (srv)    │ ◄─────────────────►  │   User2 (clt)    │
│                  │     (ngrok/CF/etc)   │                  │
│ ┌──────────────┐ │                      │ ┌──────────────┐ │
│ │ AudioStream  │ │                      │ │ AudioStream  │ │
│ │              │ │                      │ │              │ │
│ │ ┌─────────┐  │ │                      │ │ ┌─────────┐  │ │
│ │ │ Buffer  │  │ │                      │ │ │ Buffer  │  │ │
│ │ └─────────┘  │ │                      │ │ └─────────┘  │ │
│ │ ┌──────────┐ │ │                      │ │ ┌──────────┐ │ │
│ │ │Compressor│ │ │                      │ │ │Compressor│ │ │
│ │ └──────────┘ │ │                      │ │ └──────────┘ │ │
│ └──────────────┘ │                      │ └──────────────┘ │
└──────────────────┘                      └──────────────────┘
           │                                        │
           └─────────── NAT Problem ────────────────┘
               (Solved via tunneling services)
```

### How it works:

1. **User1** starts the application in server mode (`srv`)
2. **User1** tunnels their server through any service (ngrok, cloudflare, localhost.run)
3. **User1** shares the tunnel URL with **User2**
4. **User2** starts the client (`clt`) and connects to the URL
5. **Connection established** through the tunneling service, bypassing Symmetric NAT

### Core Components:

- **AudioStream**: Audio flow management (recording/playback)
- **Buffer**: Ring buffer for audio data with thread-safe operations
- **Compressor**: Dual compression (OPUS → Zstandard) for traffic optimization
- **Batch**: Batching system that packs multiple audio frames (8 frames per batch) into single packets
- **Queue**: Queue for buffering incoming audio packets
- **Server**: WebSocket server for accepting connections
- **Client**: WebSocket client for connecting to server

## 🚀 Quick Start

### Using Pre-built Releases (Recommended)

If you download pre-built releases from [GitHub Releases](https://github.com/Votline/Viz/releases), **PortAudio** and **Opus** libraries are already embedded in the binary. You don't need to install any additional dependencies - just download and use the binary.

### Building from Source

If you want to build the application yourself, you need to install system dependencies first:

#### Required System Dependencies:

- **PortAudio**: Cross-platform audio I/O library
  - Official website: [http://www.portaudio.com/](http://www.portaudio.com/)
  - GitHub: [https://github.com/PortAudio/portaudio](https://github.com/PortAudio/portaudio)
  - Installation:
    - **Linux**: 
      - `sudo apt-get install portaudio19-dev` (Debian/Ubuntu)
      - `sudo yum install portaudio-devel` (Fedora/RHEL)
      - `sudo pacman -S portaudio` (Arch Linux)
    - **macOS**: `brew install portaudio`
    - **Windows**: Download from [PortAudio downloads](http://files.portaudio.com/download.html)

- **Opus**: High-quality audio codec library
  - Official website: [https://opus-codec.org/](https://opus-codec.org/)
  - Installation:
    - **Linux**: 
      - `sudo apt-get install libopus-dev` (Debian/Ubuntu)
      - `sudo yum install opus-devel` (Fedora/RHEL)
      - `sudo pacman -S opus` (Arch Linux)
    - **macOS**: `brew install opus`
    - **Windows**: Use pre-built libraries from [Opus downloads](https://opus-codec.org/downloads/)

#### Build Steps:

1. **Clone the repository:**
```bash
git clone https://github.com/Votline/Viz
cd Viz
```

2. **Install Go dependencies:**
```bash
go mod download
```

3. **Build:**
```bash
go build -o viz main.go
```

4. **Start server:**
```bash
./viz
# Enter: server (or srv)
# Server will start on port 8443
```

5. **Tunnel the server (choose any service):**
```bash
# ngrok
ngrok http 8443

# cloudflare tunnel
cloudflared tunnel --url http://localhost:8443

# localhost.run
ssh -R 80:localhost:8443 localhost.run
```

6. **Start client:**
```bash
./viz
# Enter: client (or clt)
# Enter tunnel URL: https://your-tunnel-url.com
```

## ⚙️ Configuration

### Audio Parameters:
- **Sample Rate**: 48 kHz
- **Channels**: Mono (1 channel)
- **Bitrate**: 32 kbps
- **Buffer Size**: 2048 samples
- **Chunk Duration**: 40 ms (optimal for OPUS codec, supports 2ms-120ms range)

### Tunnel Optimization:
- **Batching**: 8 frames × 40ms = 320ms delay (optimized for tunneling services)
- **Dual Compression**: OPUS + Zstandard to minimize packet size
- **Rare Requests**: Prevents bans from tunneling services

### Network Parameters:
- **Port**: 8443
- **Read Timeout**: 28 seconds
- **Write Timeout**: 28 seconds
- **Idle Timeout**: 28 seconds

## 🔧 Technical Details

### Audio Processing:
1. **Recording**: PortAudio → Float32 → Int16 → OPUS → Zstandard → (E2EE encryption at network layer, not in audio processing chain)
2. **Playback**: (E2EE decryption at network layer) → Zstandard → OPUS → Int16 → Float32 → PortAudio

**Note**: End-to-End Encryption (E2EE) is applied at the network transport layer after audio compression, not within the audio processing pipeline itself.

### Compression (tunnel optimization):
- **OPUS**: Audio codec for voice communication (32 kbps)
- **Zstandard**: Additional compression to minimize traffic
- **Result**: Maximum compression to avoid tunnel bans

### Buffering:
- **Ring Buffers**: Circular buffers used for both recording and playback operations
  - Thread-safe operations with mutexes
  - Automatic overflow management
  - Separate read/write positions for efficient data flow
- **Batching**: Multiple compressed audio frames (8 frames × 40ms = 320ms total delay) are packed into single packets
  - Reduces WebSocket overhead
  - Creates ~320ms delay optimized for tunneling services (avoids bans)
- **Chunks**: 40ms audio chunks (optimal value for OPUS codec, which supports 2ms-120ms range)

## 📄 Licenses

### Main License
This project is distributed under the **MIT License**. See the [LICENSE](LICENSE) file for details.

### 📦 Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| [github.com/gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) | v0.0.0-20250206071425-98a94950218b | Audio I/O |
| [github.com/gorilla/websocket](https://github.com/gorilla/websocket) | v1.5.3 | WebSocket connections |
| [github.com/jj11hh/opus](https://github.com/jj11hh/opus) | v1.0.1 | OPUS audio codec |
| [go.uber.org/zap](https://go.uber.org/zap) | v1.27.0 | Structured logging |
| [github.com/klauspost/compress](https://github.com/klauspost/compress) | v1.18.1 | Zstandard compression |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | v0.43.0 | Encryption (NaCl Box) |

- **PortAudio**: MIT License - see [licenses/gordonklaus-portaudio_LICENSE.txt](licenses/gordonklaus-portaudio_LICENSE.txt)
- **Gorilla WebSocket**: BSD 2-Clause License - see [licenses/gorilla-websocket_LICENSE.txt](licenses/gorilla-websocket_LICENSE.txt)
- **Opus**: MIT License - see [licenses/hraban-opus_LICENSE.txt](licenses/hraban-opus_LICENSE.txt)
- **Uber Zap**: MIT License - see [licenses/uber-zap_LICENSE.txt](licenses/uber-zap_LICENSE.txt)
- **Klauspost Compress**: Apache 2.0 License - see [licenses/klauspost-compress_LICENSE.txt](licenses/klauspost-compress_LICENSE.txt)
- **Go Crypto (x/crypto)**: BSD 3-Clause License - see [licenses/x-crypto-nacl-box_LICENSE.txt](licenses/x-crypto-nacl-box_LICENSE.txt)

### Go Opus Authors
See [AUTHORS_opus](AUTHORS_opus) file for the list of Go Opus library authors.

---

# 🎙️ Viz - Децентрализованная P2P голосовая связь через туннели

**Viz** — это децентрализованное P2P приложение для голосовой связи, написанное на Go. Решает проблему Symmetric NAT через туннелирующие сервисы, позволяя пользователям выбирать любые серверы-посредники или использовать собственные VPS.

## ✨ Особенности

- 🌐 **Децентрализованная архитектура**: P2P связь через туннелирующие сервисы
- 🚫 **Обход NAT**: Решение проблемы Symmetric NAT
- 🔧 **Гибкость серверов**: Любые туннелирующие сервисы (ngrok, cloudflare, localhost.run)
- 🎵 **Высококачественное аудио**: OPUS кодека с битрейтом 32 кбит/с
- 📦 **Агрессивное сжатие**: OPUS + Zstandard для минимизации трафика
- ⏱️ **Оптимизированные чанки**: 40ms аудио чанки с задержкой батчей 320ms для туннелирующих сервисов
- 🔄 **Двунаправленная связь**: Одновременная запись и воспроизведение
- 🛡️ **Thread-safe**: Безопасная многопоточная обработка аудио
- 📊 **Подробное логирование**: Интеграция с Zap для мониторинга

## 🏗️ Архитектура

```
┌──────────────────┐    Tunnel Service    ┌──────────────────┐
│   User1 (srv)    │ ◄─────────────────►  │   User2 (clt)    │
│                  │     (ngrok/CF/etc)   │                  │
│ ┌──────────────┐ │                      │ ┌──────────────┐ │
│ │ AudioStream  │ │                      │ │ AudioStream  │ │
│ │              │ │                      │ │              │ │
│ │ ┌─────────┐  │ │                      │ │ ┌─────────┐  │ │
│ │ │ Buffer  │  │ │                      │ │ │ Buffer  │  │ │
│ │ └─────────┘  │ │                      │ │ └─────────┘  │ │
│ │ ┌──────────┐ │ │                      │ │ ┌──────────┐ │ │
│ │ │Compressor│ │ │                      │ │ │Compressor│ │ │
│ │ └──────────┘ │ │                      │ │ └──────────┘ │ │
│ └──────────────┘ │                      │ └──────────────┘ │
└──────────────────┘                      └──────────────────┘
           │                                        │
           └─────────── NAT Problem ────────────────┘
               (Solved via tunneling services)
```

### Как это работает:

1. **User1** запускает приложение в режиме сервера (`srv`)
2. **User1** туннелирует свой сервер через любой сервис (ngrok, cloudflare, localhost.run)
3. **User1** делится URL туннеля с **User2**
4. **User2** запускает клиент (`clt`) и подключается к URL
5. **Связь установлена** через туннелирующий сервис, обходя особенность Symmetric NAT

### Основные компоненты:

- **AudioStream**: Управление аудио потоками (запись/воспроизведение)
- **Buffer**: Кольцевой буфер для аудио данных с thread-safe операциями
- **Compressor**: Двойное сжатие (OPUS → Zstandard) для оптимизации трафика
- **Batch**: Система батчей, упаковывающая несколько аудио фреймов (8 фреймов на батч) в один пакет
- **Queue**: Очередь для буферизации входящих аудио пакетов
- **Server**: WebSocket сервер для приема соединений
- **Client**: WebSocket клиент для подключения к серверу

## 🚀 Быстрый старт

### Использование готовых релизов (Рекомендуется)

Если вы скачиваете готовые релизы с [GitHub Releases](https://github.com/Votline/Viz/releases), то библиотеки **PortAudio** и **Opus** уже встроены в бинарный файл. Вам не нужно устанавливать никаких дополнительных зависимостей - просто скачайте и используйте бинарник.

### Сборка из исходного кода

Если вы хотите собрать приложение самостоятельно, вам нужно сначала установить системные зависимости:

#### Требуемые системные зависимости:

- **PortAudio**: Кроссплатформенная библиотека для работы с аудио вводом/выводом
  - Официальный сайт: [http://www.portaudio.com/](http://www.portaudio.com/)
  - GitHub: [https://github.com/PortAudio/portaudio](https://github.com/PortAudio/portaudio)
  - Установка:
    - **Linux**: 
      - `sudo apt-get install portaudio19-dev` (Debian/Ubuntu)
      - `sudo yum install portaudio-devel` (Fedora/RHEL)
      - `sudo pacman -S portaudio` (Arch Linux)
    - **macOS**: `brew install portaudio`
    - **Windows**: Скачайте с [PortAudio downloads](http://files.portaudio.com/download.html)

- **Opus**: Высококачественная библиотека аудио кодека
  - Официальный сайт: [https://opus-codec.org/](https://opus-codec.org/)
  - Установка:
    - **Linux**: 
      - `sudo apt-get install libopus-dev` (Debian/Ubuntu)
      - `sudo yum install opus-devel` (Fedora/RHEL)
      - `sudo pacman -S opus` (Arch Linux)
    - **macOS**: `brew install opus`
    - **Windows**: Используйте готовые библиотеки с [Opus downloads](https://opus-codec.org/downloads/)

#### Шаги сборки:

1. **Клонирование репозитория:**
```bash
git clone https://github.com/Votline/Viz
cd Viz
```

2. **Установка зависимостей Go:**
```bash
go mod download
```

3. **Сборка:**
```bash
go build -o viz main.go
```

4. **Запуск сервера:**
```bash
./viz
# Введите: server (или srv)
# Сервер запустится на порту 8443
```

5. **Туннелирование сервера (выберите любой сервис):**
```bash
# ngrok
ngrok http 8443

# cloudflare tunnel
cloudflared tunnel --url http://localhost:8443

# localhost.run
ssh -R 80:localhost:8443 localhost.run
```

6. **Запуск клиента:**
```bash
./viz
# Введите: client (или clt)
# Введите URL туннеля: https://your-tunnel-url.com
```

## ⚙️ Конфигурация

### Аудио параметры:
- **Sample Rate**: 48 кГц
- **Channels**: Моно (1 канал)
- **Bitrate**: 32 кбит/с
- **Buffer Size**: 2048 сэмплов
- **Chunk Duration**: 40 мс (оптимальное значение для OPUS кодека, поддерживает диапазон 2мс-120мс)

### Оптимизация для туннелей:
- **Батчинг**: 8 фреймов × 40мс = 320мс задержка (оптимизировано для туннелирующих сервисов)
- **Двойное сжатие**: OPUS + Zstandard для минимизации размера пакетов
- **Редкие запросы**: Предотвращение банов от туннелирующих сервисов

### Сетевые параметры:
- **Port**: 8443
- **Read Timeout**: 28 секунд
- **Write Timeout**: 28 секунд
- **Idle Timeout**: 28 секунд

## 🔧 Технические детали

### Аудио обработка:
1. **Запись**: PortAudio → Float32 → Int16 → OPUS → Zstandard → (E2EE шифрование на сетевом уровне, не в цепочке обработки аудио)
2. **Воспроизведение**: (E2EE расшифровка на сетевом уровне) → Zstandard → OPUS → Int16 → Float32 → PortAudio

**Примечание**: End-to-End Encryption (E2EE) применяется на уровне сетевого транспорта после сжатия аудио, а не внутри самой цепочки обработки аудио.

### Сжатие (оптимизация для туннелей):
- **OPUS**: Аудио кодека для голосовой связи (32 кбит/с)
- **Zstandard**: Дополнительное сжатие для минимизации трафика
- **Результат**: Максимальное сжатие для избежания банов туннелей

### Буферизация:
- **Кольцевые буферы**: Круговые буферы, используемые для операций записи и воспроизведения
  - Thread-safe операции с мьютексами
  - Автоматическое управление переполнением
  - Отдельные позиции чтения/записи для эффективного потока данных
- **Батчи**: Несколько сжатых аудио фреймов (8 фреймов × 40мс = 320мс общая задержка) упаковываются в один пакет
  - Снижает накладные расходы WebSocket
  - Создаёт задержку ~320мс, оптимизированную для туннелирующих сервисов (предотвращает баны)
- **Чанки**: 40мс аудио чанки (оптимальное значение для OPUS кодека, который поддерживает диапазон 2мс-120мс)

## 📄 Лицензии

### Основная лицензия
Этот проект распространяется под лицензией **MIT License**. См. файл [LICENSE](LICENSE) для подробностей.

### 📦 Зависимости

| Пакет | Версия | Назначение |
|-------|--------|------------|
| [github.com/gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) | v0.0.0-20250206071425-98a94950218b | Аудио ввод/вывод |
| [github.com/gorilla/websocket](https://github.com/gorilla/websocket) | v1.5.3 | WebSocket соединения |
| [github.com/jj11hh/opus](https://github.com/jj11hh/opus) | v1.0.1 | OPUS аудио кодека |
| [go.uber.org/zap](https://go.uber.org/zap) | v1.27.0 | Структурированное логирование |
| [github.com/klauspost/compress](https://github.com/klauspost/compress) | v1.18.1 | Zstandard сжатие |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | v0.43.0 | Шифрование (NaCl Box) |

- **PortAudio**: MIT License - см. [licenses/gordonklaus-portaudio_LICENSE.txt](licenses/gordonklaus-portaudio_LICENSE.txt)
- **Gorilla WebSocket**: BSD 2-Clause License - см. [licenses/gorilla-websocket_LICENSE.txt](licenses/gorilla-websocket_LICENSE.txt)
- **Opus**: MIT License - см. [licenses/hraban-opus_LICENSE.txt](licenses/hraban-opus_LICENSE.txt)
- **Uber Zap**: MIT License - см. [licenses/uber-zap_LICENSE.txt](licenses/uber-zap_LICENSE.txt)
- **Klauspost Compress**: Apache 2.0 License - см. [licenses/klauspost-compress_LICENSE.txt](licenses/klauspost-compress_LICENSE.txt)
- **Go Crypto (x/crypto)**: BSD 3-Clause License - см. [licenses/x-crypto-nacl-box_LICENSE.txt](licenses/x-crypto-nacl-box_LICENSE.txt)

### Авторы OPUS библиотеки
См. файл [AUTHORS_opus](AUTHORS_opus) для списка авторов Go Opus библиотеки.
