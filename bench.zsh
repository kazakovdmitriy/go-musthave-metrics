#!/bin/zsh

# Эндпоинты с указанием метода: GET или POST
# Формат: "METHOD|URL"
ENDPOINTS=(
  "GET|http://localhost:8080/"
  "GET|http://localhost:8080/value/gauge/test"
  "POST|http://localhost:8080/update/gauge/test"
)

SHARMANKA_TIME="50s"

# Убедимся, что папка для профилей существует
mkdir -p profiles

# Массив для хранения PID'ов
pids=()

echo "Заводим шарманку на ${SHARMANKA_TIME} секунд !!!"

# Запускаем нагрузку на все эндпоинты параллельно
for entry in "${ENDPOINTS[@]}"; do
  # Разбиваем строку на метод и URL
  IFS='|' read -r method url <<< "$entry"

  if [[ "$method" == "POST" ]]; then
    # Пример тела запроса — можно изменить под твои нужды
    # Например, передаём JSON или plain text
    hey -z "${SHARMANKA_TIME}" -c 20 -m POST -d "1" -T "text/plain" "$url" &
  else
    # По умолчанию — GET
    hey -z "${SHARMANKA_TIME}" -c 20 "$url" &
  fi

  pids+=($!)
done

# Ждём 15 секунд
sleep 15

# Сохраняем heap-профиль
curl -s http://localhost:8080/debug/pprof/heap > profiles/base.pprof

# Ждём завершения всех процессов hey
for pid in "${pids[@]}"; do
  wait "$pid"
done

echo "Все нагрузочные тесты завершены."