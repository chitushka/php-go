--TEST--
Closures 002 (NEW in PHP 5.3): recursive closures via reference, closures with array_map/usort, Closure type checks
--FILE--
<?php
$fact = function ($n) use (&$fact) {
    if ($n <= 1) return 1;
    return $n * $fact($n - 1);
};
var_dump($fact(5));

$nums = array(5, 3, 1, 4, 2);
usort($nums, function ($a, $b) {
    return $a - $b;
});
var_dump($nums);

$squared = array_map(function ($x) { return $x * $x; }, array(1, 2, 3, 4));
var_dump($squared);

$people = array(
    array('name' => 'Bob', 'age' => 30),
    array('name' => 'Ann', 'age' => 25),
);
usort($people, function ($a, $b) {
    return $a['age'] - $b['age'];
});
var_dump($people[0]['name']);

// Closures are first-class values that can be stored, passed, returned
function identity($x) { return $x; }
$stored = identity(function () { return 'stored-closure'; });
var_dump($stored());

// Closures used with array_walk (by-reference callback param)
$arr = array(1, 2, 3);
array_walk($arr, function (&$v, $k) { $v = $v + $k; });
var_dump($arr);
?>
--EXPECT--
int(120)
array(5) {
  [0]=>
  int(1)
  [1]=>
  int(2)
  [2]=>
  int(3)
  [3]=>
  int(4)
  [4]=>
  int(5)
}
array(4) {
  [0]=>
  int(1)
  [1]=>
  int(4)
  [2]=>
  int(9)
  [3]=>
  int(16)
}
string(3) "Ann"
string(14) "stored-closure"
array(3) {
  [0]=>
  int(1)
  [1]=>
  int(3)
  [2]=>
  int(5)
}
