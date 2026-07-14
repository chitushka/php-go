--TEST--
P1 array elements by reference
--FILE--
<?php
$x = 1;
$items = array();
$items[0] =& $x;
$items[0] = 2;
echo $x;
echo ":";

$x = 3;
echo $items[0];
echo ":";

$alias =& $items[0];
$alias = 4;
echo $x;
echo ":";
echo $items[0];
echo ":";

$other = array('k' => 10);
$items['copy'] =& $other['k'];
$other['k'] = 11;
echo $items['copy'];
echo ":";
$items['copy'] = 12;
echo $other['k'];
echo ":";

$items[] =& $x;
$x = 5;
echo $items[1];
echo ":";

$nested = array();
$nested['inner']['value'] =& $x;
$nested['inner']['value'] = 6;
echo $x;
echo ":";
echo $items[0];
echo ":";
echo $items[1];
?>
--EXPECT--
2:3:4:4:11:12:5:6:6:6
