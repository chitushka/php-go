--TEST--
P0 list assignment non-array warning is non-halting by default
--FILE--
<?php
$a = 'old';
$b = 'old';
list($a, $b) = 42;
echo isset($a) ? 'set' : 'null';
echo ':';
echo isset($b) ? 'set' : 'null';
echo "\n";
echo 'after';
?>
--EXPECT--
null:null
after
