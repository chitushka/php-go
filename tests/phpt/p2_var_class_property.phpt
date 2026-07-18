--TEST--
P2 var class property declarations
--FILE--
<?php
class VarProperty {
    var $value = 'ok';
}

$obj = new VarProperty();
echo $obj->value;
?>
--EXPECT--
ok
