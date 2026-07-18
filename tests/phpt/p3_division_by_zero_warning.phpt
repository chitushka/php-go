--TEST--
P3 division by zero warns and returns false by default
--FILE--
<?php
$value = 5 / 0;
echo ($value === false ? 'false' : 'bad');
echo "\n";
$mod = 5 % 0.4;
echo ($mod === false ? 'mod-false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
false
mod-false
after
