--TEST--
Arrays 001: indexed/associative arrays, sorting, array functions, references in foreach
--FILE--
<?php
$a = array(3, 1, 2);
sort($a);
var_dump($a);

$b = array('b' => 2, 'a' => 1, 'c' => 3);
ksort($b);
var_dump($b);

$c = array(1, 2, 3, 4, 5);
var_dump(array_sum($c));
var_dump(array_product($c));
var_dump(array_reverse($c));
var_dump(array_slice($c, 1, 2));
var_dump(array_map(function ($x) { return $x * $x; }, array(1, 2, 3)));
var_dump(array_filter(array(1, 0, 2, null, 3, false), function ($x) { return (bool)$x; }));
var_dump(array_reduce(array(1, 2, 3, 4), function ($carry, $item) { return $carry + $item; }, 0));
var_dump(in_array(3, $c));
var_dump(array_search(3, $c));
var_dump(array_key_exists('a', $b));
var_dump(array_merge(array('a' => 1), array('a' => 2, 'b' => 3)));

$d = array(1, 2, 3);
foreach ($d as &$v) {
    $v *= 10;
}
unset($v);
var_dump($d);

$nested = array('x' => array('y' => array('z' => 42)));
var_dump($nested['x']['y']['z']);

list($p, $q, $r) = array(10, 20, 30);
var_dump($p, $q, $r);
?>
--EXPECT--
array(3) {
  [0]=>
  int(1)
  [1]=>
  int(2)
  [2]=>
  int(3)
}
array(3) {
  ["a"]=>
  int(1)
  ["b"]=>
  int(2)
  ["c"]=>
  int(3)
}
int(15)
int(120)
array(5) {
  [0]=>
  int(5)
  [1]=>
  int(4)
  [2]=>
  int(3)
  [3]=>
  int(2)
  [4]=>
  int(1)
}
array(2) {
  [0]=>
  int(2)
  [1]=>
  int(3)
}
array(3) {
  [0]=>
  int(1)
  [1]=>
  int(4)
  [2]=>
  int(9)
}
array(3) {
  [0]=>
  int(1)
  [2]=>
  int(2)
  [4]=>
  int(3)
}
int(10)
bool(true)
int(2)
bool(true)
array(2) {
  ["a"]=>
  int(2)
  ["b"]=>
  int(3)
}
array(3) {
  [0]=>
  int(10)
  [1]=>
  int(20)
  [2]=>
  int(30)
}
int(42)
int(10)
int(20)
int(30)
