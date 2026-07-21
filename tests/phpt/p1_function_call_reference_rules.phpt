--TEST--
P1 function call reference behavior and case-insensitive names
--FILE--
<?php
function MiXeDName($value) {
    return $value . '!';
}

echo mixedname('php');
echo "\n";
$fn = 'MIXEDNAME';
echo $fn('go');
echo "\n";

function bump(&$value) {
    $value = $value + 1;
}

$items = array('count' => 1);
bump($items['count']);
echo $items['count'];
echo "\n";

$target = 'n';
$n = 4;
bump($$target);
echo $n;
echo "\n";

function force($value) {
    $value = $value + 10;
}

force(&$n);
echo $n;
echo "\n";
force(&$items['count']);
echo $items['count'];
?>
--EXPECT--
php!
go!
2
5
15
12
