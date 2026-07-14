--TEST--
P1 foreach by reference
--FILE--
<?php
$items = array(1, 2, 3);
foreach ($items as &$value) {
    $value = $value * 2;
}
echo $items[0];
echo ":";
echo $items[1];
echo ":";
echo $items[2];
echo ":";

$value = 9;
echo $items[2];
echo ":";

$assoc = array('a' => 1, 'b' => 2);
foreach ($assoc as $key => &$item):
    $item = $key . $item;
endforeach;
echo $assoc['a'];
echo ":";
echo $assoc['b'];
echo ":";

$item = 'tail';
echo $assoc['b'];
?>
--EXPECT--
2:4:6:9:a1:b2:tail
