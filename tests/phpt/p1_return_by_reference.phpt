--TEST--
P1 return by reference
--FILE--
<?php
function &counter_ref() {
    static $value = 1;
    return $value;
}

$ref =& counter_ref();
echo $ref;
echo ":";
$ref = 5;
echo counter_ref();
echo ":";

$fn = 'counter_ref';
$other =& $fn();
$other = 6;
echo counter_ref();
echo ":";

$target = 'seed';
function &global_ref() {
    global $target;
    return $target;
}
$globalAlias =& global_ref();
$globalAlias = 'changed';
echo $target;
echo ":";

function &array_item_ref() {
    static $items = array('x' => 10);
    return $items['x'];
}
$item =& array_item_ref();
$item = 12;
echo array_item_ref();
?>
--EXPECT--
1:5:6:changed:12
