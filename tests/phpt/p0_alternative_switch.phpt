--TEST--
P0 alternative switch syntax
--FILE--
<?php
$value = 'b';
switch ($value):
case 'a':
    echo 'bad-a';
    break;
case 'b':
    echo 'bee';
    break;
default:
    echo 'bad-default';
endswitch;
echo ':';
$value = 'z';
switch ($value):
case 'a':
    echo 'bad-a';
    break;
default:
    echo 'default';
endswitch;
?>
--EXPECT--
bee:default
