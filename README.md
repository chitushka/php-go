# php-src-go

Начало переноса ядра PHP 5.3 на Go. Цель первого этапа - не переносить стандартные библиотеки из `ext/`, а построить минимальный фронтенд и runtime для пошагового переноса языка: lexer, parser, AST, interpreter и CLI.

## Уже реализовано

- CLI `php53`, читающий PHP-файл и выполняющий поддерживаемое подмножество.
- Лексер для `<?php`, чисел, строк, переменных, идентификаторов, `echo`, `if`, `else`, `while`, `true`, `false`, `null`.
- Парсер выражений с приоритетами для `+`, `-`, `*`, `/`, сравнений, `==`, `!=`.
- Операторы присваивания, `echo`, блоки, `if/else`, `while`.
- PHP-массивы через `array(...)` с ordered hash map, auto-index и ключами `int`/`string`.
- `foreach ($items as $value)` и `foreach ($items as $key => $value)`.
- Пользовательские функции: `function name($arg) { ... }`, вызовы, локальные frame-ы и `return`.
- Ограниченные `include` и `require` для строковых путей, разрешаемых относительно текущего PHP-файла.
- Базовые классы: `class`, `new`, методы, `__construct`, `$this`, динамические свойства и `$obj->method()`.
- Runtime со значениями `null`, `bool`, `int`, `float`, `string`, `array`, `object` и таблицей переменных.

## Запуск

```sh
go test ./...
go run ./cmd/php53 -- examples/hello.php
```

## Следующие шаги переноса

1. Добавить исключения: `throw`, `try`, `catch` и базовый объект `Exception`.
2. Подключить PHPT-тесты как acceptance suite.
