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
- Базовые классы: `class`, `extends`, `new`, методы, `__construct`, `$this`, динамические свойства, `$obj->method()`, static methods/properties.
- Модификаторы `public`, `protected`, `private`, `static` для методов и свойств, включая runtime-проверку доступа.
- `self::`, `parent::`, `static::`, late static binding для static members, а также базовые проверки `final` и `abstract`.
- `interface`, `extends` для интерфейсов, `implements` и runtime-проверка обязательных методов.
- Class constants через `const NAME = ...`, доступ `Class::NAME`, `self::NAME`, `parent::NAME` и наследование констант.
- `instanceof` для классов, родителей и интерфейсов.
- Magic methods `__get`, `__set`, `__call`, `__toString` для базового property/method overloading и строкового вывода объектов.
- Исключения: `throw`, `try`, `catch (Exception $e)` и минимальный встроенный `Exception::getMessage()`.
- PHPT-подобный runner с секциями `--TEST--`, `--FILE--`, `--EXPECT--`, `--EXPECTF--`, `--EXPECTREGEX--`, `--SKIPIF--`, `--INI--`, `--CLEAN--`, `--EXIT_CODE--`.
- Runtime со значениями `null`, `bool`, `int`, `float`, `string`, `array`, `object` и таблицей переменных.

## Запуск

```sh
go test ./...
go run ./cmd/php53 -- examples/hello.php
go run ./cmd/php53test -- tests/phpt
```

## Следующие шаги переноса

1. Добавить больше семантики выражений PHP 5.3: конкатенацию `.`, логические `&&`/`||`, индексный доступ к массивам и строкам.
2. Расширить PHPT runner поддержкой `--POST--`, `--GET--`, `--ENV--` для будущих SAPI-тестов.
