--TEST--
Smoke 005: function declarations, default args, by-reference args, variadic via func_get_args
--FILE--
<?php
function add($a, $b = 10) {
    return $a + $b;
}
var_dump(add(5));
var_dump(add(5, 1));

function increment(&$val) {
    $val++;
}
$n = 41;
increment($n);
var_dump($n);

function sumAll() {
    $args = func_get_args();
    return array_sum($args);
}
var_dump(sumAll(1, 2, 3, 4));

function makeCounter() {
    static $count = 0;
    $count++;
    return $count;
}
var_dump(makeCounter());
var_dump(makeCounter());
var_dump(makeCounter());
--EXPECT--
int(15)
int(6)
int(42)
int(10)
int(1)
int(2)
int(3)
