--TEST--
P3 property assignment on non-object warnings are non-halting by default
--FILE--
<?php
$x = 1;
$n = null;
$x->name = 'bad';
$n->name = 'bad';
echo $x;
echo "\n";
echo ($n === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
1
null
after
