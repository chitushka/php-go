--TEST--
p1 round/floor/ceil always return float, round() supports precision, abs() preserves type (BUG-5)
--FILE--
<?php
var_dump(round(3.14159, 2));
var_dump(round(1241757, -3));
var_dump(round(1.005, 2));
var_dump(round(2.5));
var_dump(round(-2.5));
var_dump(floor(3.0));
var_dump(ceil(4.0));
var_dump(abs(-5));
var_dump(abs(-10.0/2));
var_dump(abs(-5.5));
?>
--EXPECT--
float(3.14)
float(1242000)
float(1.01)
float(3)
float(-3)
float(3)
float(4)
int(5)
float(5)
float(5.5)
