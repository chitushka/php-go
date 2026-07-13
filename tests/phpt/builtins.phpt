--TEST--
core builtin functions
--FILE--
<?php
$items = array('a' => 1, 'b' => null, 'c' => 'x');
if (isset($items['a'])) {
    echo 'a';
}
if (!isset($items['b'])) {
    echo 'b';
}
if (!isset($missing)) {
    echo 'c';
}
if (empty($items['b'])) {
    echo 'd';
}
if (empty(0)) {
    echo 'e';
}
if (!empty($items['c'])) {
    echo 'f';
}
echo '|';

echo count($items);
echo ':';
echo array_key_exists('b', $items);
echo ':';
echo in_array('x', $items);
echo ':';
echo implode('-', array('a', 'b'));
echo ':';
$parts = explode(',', 'x,y,z');
echo count($parts);
echo ':';
echo $parts[1];
echo '|';

echo strlen('abcdef');
echo ':';
echo substr('abcdef', 1, 3);
echo ':';
echo substr('abcdef', -2);
echo ':';
echo strpos('abcdef', 'de');
echo ':';
echo strtoupper('php');
echo ':';
echo strtolower('GO');
--EXPECT--
abcdef|3:1:1:a-b:3:y|6:bcd:ef:3:PHP:go
