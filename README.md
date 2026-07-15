# php-src-go

Начало переноса ядра PHP 5.3 на Go. Цель первого этапа - не переносить стандартные библиотеки из `ext/`, а построить минимальный фронтенд и runtime для пошагового переноса языка: lexer, parser, AST, interpreter и CLI.

## Уже реализовано

- CLI `php53`, читающий PHP-файл и выполняющий поддерживаемое подмножество.
- CLI `php53test`, запускающий PHPT-подобные acceptance-тесты.
- Лексер/source mode для inline HTML, `<?php`, short echo `<?=`, закрывающего `?>`, комментариев, чисел, строк с базовыми escape-последовательностями, `$var` и базовым complex double-quoted interpolation (`$arr[key]`, `$obj->prop`, `${name}`, `{$arr['key']}`), heredoc/nowdoc и backticks, переменных включая variable variables `$$name`, идентификаторов, `echo`, `print`, `if`, `elseif`, `else`, `while`, `do`, `for`, `switch`, `global`, `static`, `unset`, `exit`/`die`, `true`, `false`, `null`.
- Парсер выражений с приоритетами для `+`, `-`, `*`, `/`, `%`, bitwise/shift (`&`, `|`, `^`, `~`, `<<`, `>>`), сравнений, `==`, `!=`, `<>`, `===`, `!==`, ternary `?:` и casts.
- Выражения конкатенации `.`, логические `&&`/`||` и `and`/`or` с short-circuit и PHP-приоритетом ниже присваивания, `xor`, индексный доступ `$array[$key]`, `$string[$offset]`, присваивание в массив по индексу, `$array[]` append и nested writable array targets с auto-vivification.
- Операторы присваивания как выражения, compound assignments (`+=`, `-=`, `*=`, `/=`, `%=`, `.=`), `list(...)` assignment для numeric array destructuring, expression-value semantics для `++`/`--` включая PHP-style string increment, `echo`, `print`, `unset`, `exit`/`die`, блоки, `if/elseif/else` включая alternative `if: elseif: else: endif` syntax, `while` включая alternative `while: endwhile` syntax, `do/while`, `for` включая alternative `for: endfor` syntax, `switch` включая alternative `switch: endswitch` syntax, `break`/`continue` включая numeric levels.
- PHP-массивы через `array(...)` с ordered hash map, auto-index и ключами `int`/`string`.
- `foreach ($items as $value)`, `foreach ($items as $key => $value)` и `foreach` by reference (`&$value`), включая alternative `foreach: endforeach` syntax.
- Пользовательские функции: `function name($arg = null, &$ref) { ... }`, `function &name() { ... }`, anonymous functions/closures `function (...) use (...) { ... }`, обычные вызовы и variable functions (`$fn()`) с case-insensitive lookup/redeclare checks, default parameters с базовыми compatibility checks для optional/required ordering и constant-expression defaults, pass-by-reference parameters для writable targets, return by reference, локальные frame-ы и `return`.
- References: базовое assignment by reference (`=&`) для переменных, variable variables и array elements, `foreach` by reference и return by reference с aliasing через общий variable slot.
- Переменные функций: `global $name` и persistent `static $local = ...`.
- Встроенные функции ядра: `count`, `strlen`, `substr`, `strpos`, `strtolower`, `strtoupper`, `trim`, `str_replace`, `array_key_exists`, `in_array`, `implode`, `explode`, `print_r`, `var_dump`, `round`, `floor`, `ceil`, `abs`, `min`, `max`, `time`, `date`, `file_exists`, `is_file`, `is_dir`, `basename`, `dirname`, `file_get_contents`, `file_put_contents`, `unlink`, `preg_match` с capture-массивом, `preg_replace`, `json_encode`, `json_decode`, `md5`, `sha1`, `hash`, `serialize`, `unserialize`, базовые `is_*`.
- `isset` и `empty` выполняются как language constructs: аргументы проверяются без обычного предварительного вычисления, поддержаны несколько аргументов `isset`, nested array access, свойства и static-свойства.
- Builtins вынесены в пакет `internal/runtime/builtins`; новые функции добавляются по функциональным файлам (`strings.go`, `arrays.go`, `filesystem.go`, `regex.go`, `json.go`, `hash.go`, `serialization.go`) и регистрируются в `registry.go`.
- Ограниченные `include`, `require`, `include_once` и `require_once` для строковых путей, разрешаемых относительно текущего PHP-файла.
- Базовые классы: `class`, `extends`, `new`, case-insensitive class/method lookup, `__autoload`, OOP introspection (`class_exists`, `interface_exists`, `get_class`, `get_object_vars`, `get_class_methods`, `get_class_vars`, `get_parent_class`, `class_parents`, `class_implements`, `method_exists`, `property_exists`, `is_subclass_of`, `is_a`), dynamic class names in `new`, `clone`, методы, `__construct`, old-style constructors, `__destruct`, `$this`, динамические свойства включая `$obj->$name`, `$obj->method()` и dynamic method calls (`$obj->$method()`), static methods/properties включая dynamic static member names (`Class::$$name`, `Class::$method()`).
- Модификаторы `public`, `protected`, `private`, `static` для методов и свойств, включая runtime-проверку доступа.
- `self::`, `parent::`, `static::`, late static binding для static members, `get_called_class()`, а также базовые проверки `final`, `abstract`, inherited method visibility/static/signature override и inherited property visibility/static compatibility.
- `interface`, `extends` для интерфейсов, `implements`, case-insensitive interface lookup, interface constants и runtime-проверка обязательных методов.
- Class constants через `const NAME = ...`, доступ `Class::NAME`, `self::NAME`, `parent::NAME`, `Interface::NAME` и наследование констант.
- Bare/runtime constants: bare-доступ `NAME`, `define()`, `defined()`, `constant()`, case-insensitive `define(..., true)` и fallback неизвестной bare-константы в строку имени.
- Magic constants: `__FILE__`, `__DIR__`, `__LINE__`, `__FUNCTION__`, `__CLASS__`, `__METHOD__`.
- Namespaces/imports: базовые `namespace`, `use ... as ...`, leading `\`, `__NAMESPACE__`, namespaced functions/classes/constants и fallback namespaced function/constant lookup к global имени.
- `instanceof` для классов, родителей и интерфейсов.
- Magic methods `__get`, `__set`, `__isset`, `__unset`, `__call`, `__callStatic`, `__clone`, `__sleep`, `__wakeup`, `__toString` для базового property/method overloading, клонирования, сериализации и строкового вывода объектов.
- Исключения: `throw`, `try`, несколько `catch`-блоков, пользовательские классы-наследники встроенного `Exception`, базовые `Exception::getMessage()/getCode()/getFile()/getLine()/getPrevious()/getTrace()/getTraceAsString()` и строковое представление исключения.
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

1. Следующий шаг: продолжить P2/P3 compatibility: закрывать оставшуюся OOP edge-case parity или PHP 5.3 type juggling отдельными vertical slices с PHPT-контрактом.
2. `declare`/`eval` отложены до отдельного решения; `eval` может остаться выключенным из-за security-модели.
