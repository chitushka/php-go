--TEST--
Smoke 002: basic arithmetic, precedence, integer/float division, modulo
--FILE--
<?php
var_dump(2 + 3 * 4);
var_dump((2 + 3) * 4);
var_dump(10 % 3);
var_dump(-10 % 3);
var_dump(10 / 4);
var_dump(10 / 5);
var_dump(2 - 5);
var_dump(3 == "3");
var_dump(3 === "3");
var_dump(0 == "abc");
?>
--EXPECT--
int(14)
int(20)
int(1)
int(-1)
float(2.5)
int(2)
int(-3)
bool(true)
bool(false)
bool(true)
