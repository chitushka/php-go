--TEST--
P1 function parameter compatibility defaults
--FILE--
<?php
define('DEFAULT_SUFFIX', '!');

function label($name = 'php', $suffix = DEFAULT_SUFFIX, $items = array('x' => -1, 'y' => array(2))) {
    return $name . $suffix . ':' . $items['x'] . ':' . $items['y'][0];
}

$fn = function($value = 3, $items = array('k' => 'v')) {
    return $value . $items['k'];
};

echo label();
echo "\n";
echo label('go', '?', array('x' => 7, 'y' => array(8)));
echo "\n";
echo $fn();
?>
--EXPECT--
php!:-1:2
go?:7:8
3v
