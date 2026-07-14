--TEST--
p0 array append assignment
--FILE--
<?php
$items = array('first');
$items[] = 'second';
$items[5] = 'sixth';
$items[] = 'seventh';
echo count($items);
echo ':';
echo $items[0];
echo ':';
echo $items[1];
echo ':';
echo $items[6];
$created[] = 'new';
echo ':';
echo $created[0];
$fromNull = null;
$fromNull[] = 'null';
echo ':';
echo $fromNull[0];
?>
--EXPECT--
4:first:second:seventh:new:null
