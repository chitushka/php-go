--TEST--
P2 interface constant redeclaration is rejected
--FILE--
<?php
interface DuplicateInterfaceConstant {
    const VALUE = 1;
    const VALUE = 2;
}
?>
--EXPECTF--
cannot redeclare interface constant DuplicateInterfaceConstant::VALUE
