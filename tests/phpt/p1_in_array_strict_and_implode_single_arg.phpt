--TEST--
p1 in_array() supports $strict, implode() supports single-arg and reversed-order forms (BUG-7)
--FILE--
<?php
$a = array("2", "3", "4");
var_dump(in_array(2, $a));
var_dump(in_array(2, $a, true));
var_dump(in_array("2", $a, true));

$arr = array(1, 2, 3);
var_dump(implode($arr));
var_dump(implode("-", $arr));
var_dump(implode($arr, "-"));
?>
--EXPECT--
bool(true)
bool(false)
bool(true)
string(3) "123"
string(5) "1-2-3"
string(5) "1-2-3"
