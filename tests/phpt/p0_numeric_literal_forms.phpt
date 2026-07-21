--TEST--
p0 hex, octal, exponent, and leading/trailing-dot float literals
--FILE--
<?php
var_dump(0x1A);
var_dump(0X1a);
var_dump(017);
var_dump(1e3);
var_dump(1E3);
var_dump(1.5e-3);
var_dump(1.5e+3);
var_dump(.5);
var_dump(5.);
var_dump(1 . 2);
var_dump(10);
var_dump(0);
?>
--EXPECT--
int(26)
int(26)
int(15)
float(1000)
float(1000)
float(0.0015)
float(1500)
float(0.5)
float(5)
string(2) "12"
int(10)
int(0)
