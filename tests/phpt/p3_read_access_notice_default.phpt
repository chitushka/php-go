--TEST--
P3 scalar index and property read notices are non-halting by default
--FILE--
<?php
$x = 1;
$idx = $x['key'];
$prop = $x->name;
echo ($idx === null ? 'null' : 'bad');
echo "\n";
echo ($prop === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
after
