--TEST--
p1 core predefined constants (PHP_*, M_*, E_*)
--FILE--
<?php
var_dump(PHP_INT_MAX);
var_dump(PHP_INT_SIZE);
var_dump(PHP_VERSION);
var_dump(M_PI);
var_dump(E_ALL);
var_dump(E_WARNING);
var_dump(defined('PHP_INT_MAX'));
var_dump(constant('PHP_INT_MAX'));
var_dump(define('PHP_INT_MAX', 1));
var_dump(php_int_max);
?>
--EXPECT--
int(9223372036854775807)
int(8)
string(6) "5.3.29"
float(3.1415926535898)
int(32767)
int(2)
bool(true)
int(9223372036854775807)
bool(false)
string(11) "php_int_max"
