--TEST--
p1 arrays get an independent copy on assignment, argument passing, foreach by value, array-literal embedding, object clone, and closure use() by value (BUG-9)
--FILE--
<?php
$a = array(1, 2, 3);
$b = $a;
$b[] = 99;
echo count($a) . ':' . count($b) . "\n";

function mutate($arr) {
    $arr[] = 'x';
    return $arr;
}
$c = array(1, 2);
$d = mutate($c);
echo count($c) . ':' . count($d) . "\n";

$e = array(array(1, 2), array(3, 4));
foreach ($e as $sub) {
    $sub[] = 999;
}
echo count($e[0]) . ':' . count($e[1]) . "\n";

$f = array(1, 2);
$g = array($f, 3);
$g[0][] = 'leak?';
echo count($f) . ':' . count($g[0]) . "\n";

class Holder { public $items = array(1, 2); }
$h1 = new Holder();
$h2 = clone $h1;
$h2->items[] = 'z';
echo count($h1->items) . ':' . count($h2->items) . "\n";

$src = array(1, 2);
$fn = function () use ($src) { return $src; };
$captured = $fn();
$captured[] = 'w';
echo count($src) . ':' . count($captured) . "\n";

$r1 = array(1, 2);
$r2 = &$r1;
$r2[] = 'ref-ok';
echo count($r1) . ':' . count($r2) . "\n";
?>
--EXPECT--
3:4
2:3
2:2
2:3
2:3
2:3
3:3
