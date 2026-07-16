--TEST--
P2 class constants must be constant expressions
--FILE--
<?php
class BadClassConstantValue {
    const VALUE = strlen('x');
}
?>
--EXPECTF--
class constant BadClassConstantValue::VALUE value must be a constant expression
