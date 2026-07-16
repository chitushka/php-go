--TEST--
P2 interface methods cannot be final
--FILE--
<?php
interface FinalInterfaceMethod {
    final public function value();
}
?>
--EXPECTF--
interface method FinalInterfaceMethod::value cannot be final
