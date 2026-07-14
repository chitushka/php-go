# php-src-go

Начало переноса ядра PHP 5.3 на Go. Цель первого этапа - не переносить стандартные библиотеки из `ext/`, а построить минимальный фронтенд и runtime для пошагового переноса языка: lexer, parser, AST, interpreter и CLI.

## Уже реализовано

- CLI `php53`, читающий PHP-файл и выполняющий поддерживаемое подмножество.
- CLI `php53test`, запускающий PHPT-подобные acceptance-тесты.
- Лексер для `<?php`, закрывающего `?>`, комментариев, чисел, строк, переменных, идентификаторов, `echo`, `print`, `if`, `elseif`, `else`, `while`, `do`, `for`, `switch`, `global`, `static`, `unset`, `exit`/`die`, `true`, `false`, `null`.
- Парсер выражений с приоритетами для `+`, `-`, `*`, `/`, `%`, bitwise/shift (`&`, `|`, `^`, `~`, `<<`, `>>`), сравнений, `==`, `!=`, `<>`, `===`, `!==`, ternary `?:` и casts.
- Выражения конкатенации `.`, логические `&&`/`||` и `and`/`or` с short-circuit и PHP-приоритетом ниже присваивания, `xor`, индексный доступ `$array[$key]`, `$string[$offset]`, присваивание в массив по индексу и `$array[]` append.
- Операторы присваивания как выражения, compound assignments (`+=`, `-=`, `*=`, `/=`, `%=`, `.=`), statement-level `++`/`--`, `echo`, `print`, `unset`, `exit`/`die`, блоки, `if/elseif/else`, `while`, `do/while`, `for`, `switch`, `break`, `continue`.
- PHP-массивы через `array(...)` с ordered hash map, auto-index и ключами `int`/`string`.
- `foreach ($items as $value)` и `foreach ($items as $key => $value)`.
- Пользовательские функции: `function name($arg = null, &$ref) { ... }`, вызовы, default parameters, pass-by-reference parameters, локальные frame-ы и `return`.
- Переменные функций: `global $name` и persistent `static $local = ...`.
- Встроенные функции ядра: `isset`, `empty`, `count`, `strlen`, `substr`, `strpos`, `strtolower`, `strtoupper`, `trim`, `str_replace`, `array_key_exists`, `in_array`, `implode`, `explode`, `print_r`, `var_dump`, `round`, `floor`, `ceil`, `abs`, `min`, `max`, `time`, `date`, `file_exists`, `is_file`, `is_dir`, `basename`, `dirname`, `file_get_contents`, `file_put_contents`, `unlink`, `preg_match` с capture-массивом, `preg_replace`, `json_encode`, `json_decode`, `md5`, `sha1`, `hash`, `serialize`, `unserialize`, базовые `is_*`.
- Builtins вынесены в пакет `internal/runtime/builtins`; новые функции добавляются по функциональным файлам (`strings.go`, `arrays.go`, `filesystem.go`, `regex.go`, `json.go`, `hash.go`, `serialization.go`) и регистрируются в `registry.go`.
- Ограниченные `include` и `require` для строковых путей, разрешаемых относительно текущего PHP-файла.
- Базовые классы: `class`, `extends`, `new`, методы, `__construct`, `$this`, динамические свойства, `$obj->method()`, static methods/properties.
- Модификаторы `public`, `protected`, `private`, `static` для методов и свойств, включая runtime-проверку доступа.
- `self::`, `parent::`, `static::`, late static binding для static members, а также базовые проверки `final` и `abstract`.
- `interface`, `extends` для интерфейсов, `implements` и runtime-проверка обязательных методов.
- Class constants через `const NAME = ...`, доступ `Class::NAME`, `self::NAME`, `parent::NAME` и наследование констант.
- `instanceof` для классов, родителей и интерфейсов.
- Magic methods `__get`, `__set`, `__call`, `__toString` для базового property/method overloading и строкового вывода объектов.
- Исключения: `throw`, `try`, `catch (Exception $e)` и минимальный встроенный `Exception::getMessage()`.
- PHPT-подобный runner с секциями `--TEST--`, `--FILE--`, `--EXPECT--`, `--EXPECTF--`, `--EXPECTREGEX--`, `--SKIPIF--`, `--INI--`, `--CLEAN--`, `--EXIT_CODE--`, `--GET--`, `--POST--`, `--ENV--`, `--COOKIE--`, `--HEADERS--`.
- SAPI-модель PHPT runner подготавливает `$_GET`, `$_POST`, `$_COOKIE`, `$_FILES`, `$_REQUEST`, `$_ENV`, `$_SERVER` с базовыми request/server/header полями; поддержаны синтетический `--FILES--`, `--POST_RAW--` multipart/form-data, nested upload names (`file[]`, `file[group][]`, `file[avatar][front]`), upload INI-лимиты (`post_max_size`, `upload_max_filesize`, `file_uploads`, `max_file_uploads`, `upload_tmp_dir`), RFC1867 `MAX_FILE_SIZE`/empty filename edge cases, `is_uploaded_file()` и `move_uploaded_file()`.
- Response headers: runtime поддерживает `header()` и `headers_list()`.
- Runtime со значениями `null`, `bool`, `int`, `float`, `string`, `array`, `object` и таблицей переменных.

## Запуск

```sh
go test ./...
go run ./cmd/php53 examples/hello.php
go run ./cmd/php53test tests/phpt
```

## Следующие шаги переноса

1. Продолжить P0-языковые конструкции из `AGENTS.md`: inline HTML/short echo, string escapes/interpolation, alternative control syntax, `list(...)`, `include_once`/`require_once` и nested writable targets.
