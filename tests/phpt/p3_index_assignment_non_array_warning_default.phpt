--TEST--
P3 index assignment on non-array warnings are non-halting by default
--FILE--
<?php
$x = 1;
$name = 'y';
$y = 'scalar';
$nested = array('a' => 1);
$x['key'] = 'bad';
$$name['key'] = 'bad';
$nested['a']['b'] = 'bad';
echo $x;
echo "\n";
echo $y;
echo "\n";
echo $nested['a'];
echo "\n";
echo 'after';
?>
--EXPECT--
1
scalar
1
after
