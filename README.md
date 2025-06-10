# Worker Pool на Go

Расширяемый пул воркеров с динамическим масштабированием и поддержкой middleware.

---

## ✨ Возможности

- **Динамические воркеры**  
  Добавление/удаление воркеров в любое время (`AddWorker()`, `RemoveWorker(id)`).
- **Авто-масштабирование**  
  Автоматическое увеличение/понижение количества воркеров по длине очереди.
- **Пайплайн middleware**  
  Обёртывайте задачи логированием, восстановлением после паник, ретраев, рейт лимитов или собственными перехватчиками.
- **Контекст-менеджмент**  
  `SubmitContext(ctx, job)` позволяет отменять или задавать таймаут для отдельных задач.
- **Встроенный логгер**  
  Каждый воркер использует `log.Logger` из стандартной библиотеки — без внешних зависимостей.

---

## 🚀 Пример 

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/Miraines/vk-internship/internal/pool"
)

func main() {
    p := pool.New(
        50,
        pool.WithInitialWorkers(3),
        pool.WithLogger(log.Default()),
        pool.WithMiddleware(
            pool.LoggingMiddleware(nil),
            pool.RecoverMiddleware(nil),
            pool.RetryMiddleware(2),
        ),
        pool.WithAutoScaler(pool.AutoScaleConfig{
            Min:          1,
            Max:          8,
            UpThreshold:  5,
            ObserveEvery: time.Second,
        }),
    )
    defer p.Shutdown()

    // Отправляем задачи
    for i := 1; i <= 20; i++ {
        p.SubmitContext(context.Background(), fmt.Sprintf("job-%02d", i))
    }

    time.Sleep(3 * time.Second)
}
```

## 📁 Структура проекта

```
vk-internship/
├── cmd/demo/main.go
└── internal/pool/
    ├── pool.go
    ├── options.go
    ├── middleware.go
    └── autoscale.go
```

<p align="center"><sub>© 2025 Miraines • MIT License</sub></p>
