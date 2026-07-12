# php-src-go

Начало переноса ядра PHP 5.3 на Go. Цель первого этапа - не переносить стандартные библиотеки из `ext/`, а построить минимальный фронтенд и runtime для пошагового переноса языка: lexer, parser, AST, interpreter и CLI.

## Уже реализовано

- CLI `php53`, читающий PHP-файл и выполняющий поддерживаемое подмножество.
- Лексер для `<?php`, чисел, строк, переменных, идентификаторов, `echo`, `if`, `else`, `while`, `true`, `false`, `null`.
- Парсер выражений с приоритетами для `+`, `-`, `*`, `/`, сравнений, `==`, `!=`.
- Операторы присваивания, `echo`, блоки, `if/else`, `while`.
- PHP-массивы через `array(...)` с ordered hash map, auto-index и ключами `int`/`string`.
- `foreach ($items as $value)` и `foreach ($items as $key => $value)`.
- Runtime со значениями `null`, `bool`, `int`, `float`, `string`, `array` и таблицей переменных.

## Запуск

```sh
go test ./...
go run ./cmd/php53 -- examples/hello.php
```

## Следующие шаги переноса

1. Добавить функции, call frames и `return`.
2. Добавить include/require в ограниченном статическом виде.
3. Добавить классы, объекты, методы и исключения.
4. Подключить PHPT-тесты как acceptance suite.
