#!/bin/zsh

# Проверяем, передан ли аргумент
if [[ $# -ne 1 ]] || [[ "$1" != "base" && "$1" != "result" ]]; then
  echo "Использование: $0 {base|result}"
  exit 1
fi

PROFILE_NAME="$1"

# Эндпоинты с указанием метода: GET или POST
# Формат: "METHOD|URL"
ENDPOINTS=(
  "GET|http://localhost:8080/"
  "POST|http://localhost:8080/update/gauge/test/123"
  "GET|http://localhost:8080/value/gauge/test"
)

SHARMANKA_TIME="30s"

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
    hey -z "${SHARMANKA_TIME}" -c 20 -m POST "$url" &
  else
    hey -z "${SHARMANKA_TIME}" -c 20 "$url" &
  fi

  pids+=($!)
done

# Ждём 15 секунд
sleep 15

# Сохраняем heap-профиль с нужным именем
curl -s "http://localhost:6060/debug/pprof/heap?seconds=1" > "profiles/${PROFILE_NAME}.pprof"

# Ждём завершения всех процессов hey
for pid in "${pids[@]}"; do
  wait "$pid"
done

echo "Все нагрузочные тесты завершены. Профиль сохранён как profiles/${PROFILE_NAME}.pprof"