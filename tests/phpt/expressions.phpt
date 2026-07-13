--TEST--
concat logical and index expressions
--FILE--
<?php
class Label {
    public function __toString() {
        return 'B';
    }
}

$label = new Label();
echo 'A' . $label . 3;
echo '|';

$items = array('name' => 'php', 2 => 'go');
echo $items['name'];
echo ':';
echo $items[2];
$items['name'] = 'zend';
echo ':';
echo $items['name'];
echo '|';

$s = 'php';
echo $s[0];
echo $s[1];
echo $s[2];
echo '|';

$x = 0;
function fail() {
    return 1 / 0;
}
if ($x && fail()) {
    echo 'bad';
} else {
    echo 'and';
}
if (1 || fail()) {
    echo 'or';
}
--EXPECT--
AB3|php:go:zend|php|andor
