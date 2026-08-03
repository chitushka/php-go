--TEST--
P0 simple interpolation syntax does not chain array/property access
--FILE--
<?php
class Box {
    public $a = 'value';
}

$arr = array('a' => array('b' => 'X'));
$obj = new Box();

echo "$arr[a][b]";
echo "\n";
echo "$obj->a->b";
?>
--EXPECT--
Array[b]
value->b
