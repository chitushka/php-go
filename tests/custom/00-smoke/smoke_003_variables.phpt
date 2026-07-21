--TEST--
Smoke 003: variables, references, variable variables, isset/empty/unset
--FILE--
<?php
$a = 1;
$b = &$a;
$b = 2;
var_dump($a);

$name = 'foo';
$$name = 'bar';
var_dump($foo);

$c = 5;
unset($c);
var_dump(isset($c));
var_dump(empty($c));

$d = 0;
var_dump(isset($d));
var_dump(empty($d));

$e = null;
var_dump(isset($e));
var_dump(is_null($e));
?>
--EXPECT--
int(2)
string(3) "bar"
bool(false)
bool(true)
bool(true)
bool(true)
bool(false)
bool(true)
