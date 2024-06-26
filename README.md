# Оценка времени оставшегося до окончания выполнения catchup

## Методология

Для оценки времени, необходимого для завершения операции catchup, нам необходимо учесть несколько ключевых моментов (метрик):

1. **Извлечение времени создания резервной копии**: Это позволяет нам определить точку отсчета для поиска WAL файлов.
2. **Получение списка WAL файлов**: Нам нужно найти все WAL файлы, созданные после создания резервной копии.
3. **Оценка времени применения одного WAL файла**: На основе исторических данных или фиксированного значения.
4. **Установление соединения и сбор информации о системе**: Сравнение системных идентификаторов и получение информации о контрольных точках (LSN).
5. **Оценка общего времени catchup**: На основе количества WAL файлов и времени, необходимого для применения каждого файла.

## Логика работы функций

### Функция `execCommand`

Выполняет команду с неопределённым количеством передаваемых параметров и возвращает результат в виде строки. Это удобно для выполнения командной строки и получения её вывода.

### Функция `getBackupCreationTime`

Выполняет команду `wal-g backup-list --detail` для получения списка резервных копий. Разбирает вывод команды и находит нужную резервную копию по имени. Извлекает и возвращает время создания резервной копии.

### Функция `getWALFilesSince`

Выполняет команду `wal-g wal-show` для получения списка WAL файлов. Разбирает вывод команды и собирает список WAL файлов, созданных после указанного времени резервной копии.

### Функция `estimateWALApplyTime`

Возвращает фиксированное среднее время применения одного WAL файла (для упрощения задачи). В реальном сценарии можно использовать статистические данные, полученные путём сбора логов с BD.

### Функция `establishConnection`

Устанавливает соединение с BD. Получает информацию о контрольной точке и сравнивает её с текущей системой. Проверяет совпадение системных идентификаторов и временных линий.

### Функция `estimateCatchupTime`

Использует предыдущие функции для получения времени создания резервной копии, списка WAL файлов и установления соединения. Получает текущий LSN и сравнивает его с контрольной точкой. Рассчитывает общее время для применения всех WAL файлов на основе их количества и среднего времени применения одного файла.


