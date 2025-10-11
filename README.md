# WB Orders Service

Демонстрационный микросервис для работы с заказами.

## Технологии

- **Go** - язык программирования
- **PostgreSQL** - база данных
- **NATS Streaming** - брокер сообщений
- **Docker** - контейнеризация NATS
- **In-memory cache** - кэширование данных

## Функциональность

- Подписка на канал NATS Streaming
- Сохранение заказов в PostgreSQL
- Кэширование данных в памяти
- Восстановление кэша из БД при перезапуске
- HTTP API для получения заказов по ID
- Веб-интерфейс для просмотра заказов


## Установка и запуск

### Требования

- Go 1.21+
- PostgreSQL 18
- Docker Desktop

### 1. Клонирование репозитория

```bash
git clone https://github.com/ВАШ_USERNAME/wb-orders-service.git
cd wb-orders-service

### 2. Установка зависимостей
go mod download

### 3. Настройка PostgreSQL
Создайте базу данных и пользователя

CREATE USER wb_user WITH PASSWORD 'wb_password';
CREATE DATABASE wb_orders OWNER wb_user;

### 4. Запуск NATS Streaming
docker-compose up -d

### 5. Запуск приложения
go run cmd/service/main.go

### 6. Отправка тестового заказа
go run scripts/publisher.go

### 7. Открыть веб-интерфейс
http://localhost:8080

Получить заказ по ID
GET /api/order/{order_uid}

Пример: GET http://localhost:8080/api/order/b563feb7b2b84b6test

Автор Vad1m03