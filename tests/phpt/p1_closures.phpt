--TEST--
P1 anonymous functions and closures
--FILE--
<?php
$base = 10;
$add = function($value) use ($base) {
    return $base + $value;
};
$base = 100;
echo $add(5);
echo ":";

$counter = 1;
$bump = function() use (&$counter) {
    $counter = $counter + 1;
    return $counter;
};
echo $bump();
echo ":";
echo $counter;
echo ":";

$prefix = 'go';
$join = function($suffix = '!') use ($prefix) {
    return $prefix . $suffix;
};
echo $join('?');
echo ":";

$items = array('value' => 3);
$ref = function &() use (&$items) {
    return $items['value'];
};
$alias =& $ref();
$alias = 7;
echo $items['value'];
?>
--EXPECT--
15:2:2:go?:7
