--TEST--
p3 intval() with default and explicit $base
--FILE--
<?php
var_dump(intval("1e3"));
var_dump(intval(42));
var_dump(intval(4.9));
var_dump(intval(true));
var_dump(intval(null));
var_dump(intval("  42abc"));
var_dump(intval("-42"));
var_dump(intval("0x1A", 16));
var_dump(intval("1A", 16));
var_dump(intval("0x1A", 0));
var_dump(intval("012", 0));
var_dump(intval("111", 2));
?>
--EXPECT--
int(1000)
int(42)
int(4)
int(1)
int(0)
int(42)
int(-42)
int(26)
int(26)
int(26)
int(10)
int(7)
