--TEST--
p1 int arithmetic overflow promotes to float instead of wrapping (BUG-4)
--FILE--
<?php
var_dump(9223372036854775807 + 1);
var_dump(PHP_INT_MAX + 1);
?>
--EXPECT--
float(9.2233720368548E+18)
float(9.2233720368548E+18)
