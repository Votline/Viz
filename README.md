# 🎙️ Viz - Decentralized P2P Voice Communication

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![WebSocket](https://img.shields.io/badge/Protocol-WebSocket-orange.svg)](https://tools.ietf.org/html/rfc6455)
[![Audio Codec](https://img.shields.io/badge/Codec-OPUS-red.svg)](https://opus-codec.org/)
[![Architecture](https://img.shields.io/badge/Architecture-P2P%20via%20Tunnels-purple.svg)]()

**Viz** — это децентрализованное P2P приложение для голосовой связи, написанное на Go. Решает проблему Symmetric NAT через туннелирующие сервисы, позволяя пользователям выбирать любые серверы-посредники или использовать собственные VPS.

## ✨ Особенности

- 🌐 **Децентрализованная архитектура**: P2P связь через туннелирующие сервисы
- 🚫 **Обход NAT**: Решение проблемы Symmetric NAT
- 🔧 **Гибкость серверов**: Любые туннелирующие сервисы (ngrok, cloudflare, localhost.run)
- 🎵 **Высококачественное аудио**: OPUS кодека с битрейтом 32 кбит/с
- 📦 **Агрессивное сжатие**: OPUS + Zstandard для минимизации трафика
- ⏱️ **Оптимизированные чанки**: 300ms пакеты для избежания банов туннелей
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
- **Queue**: Очередь для буферизации входящих аудио пакетов
- **Server**: WebSocket сервер для приема соединений
- **Client**: WebSocket клиент для подключения к серверу

## 🚀 Быстрый старт

### Сборка и запуск

1. **Клонирование репозитория:**
```bash
git clone https://github.com/Votline/Viz
cd Viz
```

2. **Установка зависимостей:**
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
- **Chunk Duration**: 300 мс (оптимизировано для туннелей)

### Оптимизация для туннелей:
- **Большие чанки**: 300ms вместо стандартных 20ms для снижения частоты запросов
- **Двойное сжатие**: OPUS + Zstandard для минимизации размера пакетов
- **Редкие запросы**: Предотвращение банов от туннелирующих сервисов

### Сетевые параметры:
- **Port**: 8443
- **Read Timeout**: 28 секунд
- **Write Timeout**: 28 секунд
- **Idle Timeout**: 28 секунд

## 🔧 Технические детали

### Аудио обработка:
1. **Запись**: PortAudio → Float32 → Int16 → OPUS → Zstandard
2. **Воспроизведение**: Zstandard → OPUS → Int16 → Float32 → PortAudio

### Сжатие (оптимизация для туннелей):
- **OPUS**: Аудио кодека для голосовой связи (32 кбит/с)
- **Zstandard**: Дополнительное сжатие для минимизации трафика
- **Результат**: Максимальное сжатие для избежания банов туннелей

### Буферизация:
- Кольцевые буферы для записи и воспроизведения
- Thread-safe операции с мьютексами
- Автоматическое управление переполнением
- Оптимизированные 300ms чанки для туннелирующих сервисов

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

- **PortAudio**: MIT License - см. [licenses/gordonklaus-portaudio_LICENSE.txt](licenses/gordonklaus-portaudio_LICENSE.txt)
- **Gorilla WebSocket**: BSD 2-Clause License - см. [licenses/gorilla-websocket_LICENSE.txt](licenses/gorilla-websocket_LICENSE.txt)
- **Go Opus**: MIT License - см. [licenses/go-opus_LICENSE.txt](licenses/go-opus_LICENSE.txt)
- **Uber Zap**: MIT License - см. [licenses/uber-zap_LICENSE.txt](licenses/uber-zap_LICENSE.txt)
- **Klauspost Compress**: Apache 2.0 License - см. [licenses/klauspost-compress_LICENSE.txt](licenses/klauspost-compress_LICENSE.txt)

### Авторы OPUS библиотеки
См. файл [AUTHORS_opus](AUTHORS_opus) для списка авторов Go Opus библиотеки.
