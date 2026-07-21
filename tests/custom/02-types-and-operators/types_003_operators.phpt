--TEST--
Types 003: ternary shorthand (NEW in 5.3), logical vs boolean operator precedence, bitwise ops, string increment
--FILE--
<?php
// Ternary shorthand $a ?: $b was introduced in PHP 5.3.0
$a = 0;
$b = $a ?: 'default';
var_dump($b);
$c = 'value';
$d = $c ?: 'default';
var_dump($d);

// and/or have lower precedence than =, unlike && / ||
$e = false or true;
var_dump($e); // bool(false): equivalent to ($e = false) or true

$f = false || true;
var_dump($f); // bool(true): equivalent to $f = (false || true)

// bitwise
var_dump(5 & 3);
var_dump(5 | 2);
var_dump(5 ^ 1);
var_dump(~5);
var_dump(1 << 3);
var_dump(16 >> 2);

// string increment (perl-style)
$s = 'a';
$s++;
var_dump($s);
$s = 'z';
$s++;
var_dump($s);
$s = 'az';
$s++;
var_dump($s);
$s = 'zz';
$s++;
var_dump($s);
?>
--EXPECT--
string(7) "default"
string(5) "value"
bool(false)
bool(true)
int(1)
int(7)
int(4)
int(-6)
int(8)
int(4)
string(1) "b"
string(2) "aa"
string(2) "ba"
string(3) "aaa"
