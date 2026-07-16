--TEST--
P2 class constant redeclaration is rejected
--FILE--
<?php
class DuplicateClassConstant {
    const VALUE = 1;
    const VALUE = 2;
}
?>
--EXPECTF--
cannot redeclare class constant DuplicateClassConstant::VALUE
