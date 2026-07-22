--TEST--
p1 arithmetic +/-/* and unary minus/plus preserve float type on whole-number results (BUG-6)
--FILE--
<?php
var_dump(5.0 + 3.0);
var_dump(5.0 + 3);
var_dump(2.0 * 3);
var_dump(5.0 - 2.0);
var_dump(-5.0);
var_dump(+5.0);
var_dump(1 + "10 apples");
var_dump("5" + "3");
var_dump("5.0" + "3");
var_dump(6 / 2);
var_dump(7 / 2);
?>
--EXPECT--
float(8)
float(8)
float(6)
float(3)
float(-5)
float(5)
int(11)
int(8)
float(8)
int(3)
float(3.5)
