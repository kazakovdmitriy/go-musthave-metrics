# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Оптимизация памяти
Оптимизировал мидлварь и сервис gzip, и mainpageservice

1. Использовал sync.Pool для компрессии/декомпрессии, что позволило переиспользовать объекты вместо создания новых на каждый запрос
```go
gzipReaderPool = sync.Pool{
    New: func() interface{} {
        return new(gzip.Reader)
    },
}

gzipWriterPool = sync.Pool{
New: func() interface{} {
return gzip.NewWriter(io.Discard)
},
}
```

2. В main_page_service и dbstorage использовал strings.Builder вместо fmt.Sprintf 
```go
// Было:
result += fmt.Sprintf("<li>%s = %d</li>\n", id, delta.Int64)

// Стало:
builder.WriteString("<li>")
builder.WriteString(id)
builder.WriteString(" = ")
buf := strconv.AppendInt(make([]byte, 0, 20), delta.Int64, 10)
builder.Write(buf)
```

3. В main_page_service использовал template для кеширования шаблонов

```go
template.New("mainpage").Parse(...)
```

Получил следующие результаты:

| flat      | flat%   | sum%   | cum       | cum%   | function                                                              |
|-----------|---------|--------|-----------|--------|-----------------------------------------------------------------------|
| -3610.34kB| 35.52%  | 35.52% | -3615.04kB| 35.57% | `compress/flate.NewWriter` (inline)                                   |
| 2064.04kB | 20.31%  | 15.21% | 2064.04kB | 20.31% | `github.com/jackc/chunkreader/v2.(*ChunkReader).newBuf` (inline)      |
| -1026.25kB| 10.10%  | 25.31% | -1026.25kB| 10.10% | `compress/flate.(*huffmanEncoder).generate`                           |
| 1026kB    | 10.09%  | 15.22% | 1026kB    | 10.09% | `bufio.NewWriterSize` (inline)                                        |
| -514.63kB | 5.06%   | 20.28% | -514.63kB | 5.06%  | `github.com/jackc/pgx/v4/pgxpool.(*connResource).getPoolRow` (inline) |
| 514.63kB  | 5.06%   | 15.22% | 514.63kB  | 5.06%  | `github.com/jackc/pgx/v4/pgxpool.(*connResource).getPoolRows` (inline)|
| 512.75kB  | 5.04%   | 10.17% | 3603.47kB | 35.45% | `github.com/kazakovdmitriy/go-musthave-metrics/internal/repository/dbstorage.(*dbstorage).GetAllMetrics.func1` |
| 512.69kB  | 5.04%   | 5.13%  | 512.69kB  | 5.04%  | `strings.(*Builder).grow`                                             |
| -512.12kB | 5.04%   | 10.17% | -590.76kB | 5.81%  | `github.com/kazakovdmitriy/go-musthave-metrics/internal/handler.setupMiddlewares.HashValidationMiddleware.func6.1` |
| 512.12kB  | 5.04%   | 5.13%  | 512.12kB  | 5.04%  | `strings.(*Builder).Write`                                            |
| 512.12kB  | 5.04%   | 0.089% | 512.12kB  | 5.04%  | `github.com/jackc/pgx/v4.(*Conn).getRows`                             |
| -512.12kB | 5.04%   | 5.13%  | 513.88kB  | 5.06%  | `net/http.(*conn).readRequest`                                        |
| -512.09kB | 5.04%   | 10.17% | -527.74kB | 5.19%  | `github.com/kazakovdmitriy/go-musthave-metrics/internal/handler.setupMiddlewares.RequestLogger.func1.1` |
| 512.05kB  | 5.04%   | 5.13%  | 512.05kB  | 5.04%  | `github.com/jackc/puddle.(*Pool).Acquire`                             |
| 512.04kB  | 5.04%   | 0.09%  | 512.04kB  | 5.04%  | `compress/flate.newHuffmanEncoder` (inline)                           |
| -512.03kB | 5.04%   | 5.13%  | -512.03kB | 5.04%  | `internal/profile.init.func4`                                         |
| 512.01kB  | 5.04%   | 0.09%  | 509.50kB  | 5.01%  | `github.com/kazakovdmitriy/go-musthave-metrics/internal/repository/dbstorage.(*dbstorage).GetGauge.func1` |
| -4.18kB   | 0.041%  | 0.13%  | -4.18kB   | 0.041% | `compress/flate.(*compressor).initDeflate` (inline)                   |

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**
