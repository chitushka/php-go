--TEST--
array and foreach
--FILE--
<?php
$items = array('x' => 7, 3 => 'z', 'last');
foreach ($items as $key => $value) {
    echo $key;
    echo ':';
    echo $value;
    echo ';';
}
--EXPECT--
x:7;3:z;4:last;
