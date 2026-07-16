--TEST--
P2 interface constants must be constant expressions
--FILE--
<?php
interface BadInterfaceConstantValue {
    const VALUE = strlen('x');
}
?>
--EXPECTF--
interface constant BadInterfaceConstantValue::VALUE value must be a constant expression
