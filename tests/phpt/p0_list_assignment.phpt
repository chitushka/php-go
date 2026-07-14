--TEST--
P0 list assignment
--FILE--
<?php
$values = array('first', 'skip', array('nested-a', 'nested-b'));
list($a, , list($b, $c)) = $values;
echo $a;
echo ':';
echo $b;
echo ':';
echo $c;
echo ':';
list($x, $y, $z) = array(1);
echo $x;
echo ':';
if (!isset($y) && !isset($z)) {
    echo 'missing';
}
?>
--EXPECT--
first:nested-a:nested-b:1:missing
