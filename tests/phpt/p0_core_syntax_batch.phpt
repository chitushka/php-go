--TEST--
p0 core syntax batch
--FILE--
<?php
// line comment
/* block comment */
$a = 5;
$a += 3;
$a %= 5;
$a++;
++$a;
$a -= 1;
$a *= 3;
$a /= 2;
echo $a;
echo ':';
$s = 'php';
$s .= '-go';
print $s;
echo ':';
if ($a === 6) {
    echo 'strict';
} elseif ($a <> 6) {
    echo 'bad';
} else {
    echo 'bad';
}
$items = array('x' => 1, 'y' => 2);
unset($items['x']);
if (!isset($items['x'])) {
    echo ':unset';
}
?>
--EXPECT--
6:php-go:strict:unset
