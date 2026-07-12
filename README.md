# php-src-go

Начало переноса ядра PHP 5.3 на Go. Цель первого этапа - не переносить стандартные библиотеки из `ext/`, а построить минимальный фронтенд и runtime для пошагового переноса языка: lexer, parser, AST, interpreter и CLI.

## Уже реализовано

- CLI `php53`, читающий PHP-файл и выполняющий поддерживаемое подмножество.
- CLI `php53test`, запускающий PHPT-подобные acceptance-тесты.
- Лексер для `<?php`, чисел, строк, переменных, идентификаторов, `echo`, `if`, `else`, `while`, `true`, `false`, `null`.
- Парсер выражений с приоритетами для `+`, `-`, `*`, `/`, сравнений, `==`, `!=`.
- Операторы присваивания, `echo`, блоки, `if/else`, `while`.
- PHP-массивы через `array(...)` с ordered hash map, auto-index и ключами `int`/`string`.
- `foreach ($items as $value)` и `foreach ($items as $key => $value)`.
- Пользовательские функции: `function name($arg) { ... }`, вызовы, локальные frame-ы и `return`.
- Ограниченные `include` и `require` для строковых путей, разрешаемых относительно текущего PHP-файла.
- Базовые классы: `class`, `new`, методы, `__construct`, `$this`, динамические свойства и `$obj->method()`.
- Исключения: `throw`, `try`, `catch (Exception $e)` и минимальный встроенный `Exception::getMessage()`.
- PHPT-подобный runner с секциями `--TEST--`, `--FILE--`, `--EXPECT--`.
- Runtime со значениями `null`, `bool`, `int`, `float`, `string`, `array`, `object` и таблицей переменных.

## Запуск

```sh
go test ./...
go run ./cmd/php53 -- examples/hello.php
go run ./cmd/php53test -- tests/phpt
```

## Следующие шаги переноса

1. Расширить PHPT runner секциями `--SKIPIF--`, `--EXPECTF--`, `--INI--`.
2. Расширить объектную модель: `public/private/protected`, `extends`, static members.
