--TEST--
P3 array loose and strict comparisons
--FILE--
<?php
$a = array('x' => 1, 'y' => '2');
$b = array('y' => 2, 'x' => '1');
$sameOrder = array('x' => 1, 'y' => '2');
$differentValue = array('x' => 1, 'y' => 3);
$short = array('x' => 1);
$empty = array();
$one = array(1);

echo ($a == $b ? 'eq' : 'bad');
echo ':';
echo ($a === $sameOrder ? 'ident' : 'bad');
echo ':';
echo ($a === $b ? 'bad' : 'not-ident');
echo ':';
echo ($a != $differentValue ? 'diff' : 'bad');
echo "\n";
echo ($short < $a ? 'short' : 'bad');
echo ':';
echo ($differentValue > $a ? 'greater' : 'bad');
echo ':';
echo ($empty == null ? 'empty-null' : 'bad');
echo ':';
echo ($one > 0 ? 'array-greater' : 'bad');
?>
--EXPECT--
eq:ident:not-ident:diff
short:greater:empty-null:array-greater
