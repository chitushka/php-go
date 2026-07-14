--TEST--
P0 isset and empty construct semantics
--FILE--
<?php
$items = array('a' => 1, 'b' => null, 'c' => array('d' => '0', 'e' => 'x'));
if (isset($items['a'], $items['c']['e'])) { echo 'a'; }
if (!isset($items['a'], $items['b'])) { echo 'b'; }
if (!isset($missing['x'])) { echo 'c'; }
if (empty($items['b'])) { echo 'd'; }
if (empty($items['c']['d'])) { echo 'e'; }
if (!empty($items['c']['e'])) { echo 'f'; }
if (empty($missing['x'])) { echo 'g'; }
?>
--EXPECT--
abcdefg
