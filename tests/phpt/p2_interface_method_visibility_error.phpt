--TEST--
P2 interface methods must be public
--FILE--
<?php
interface HiddenInterfaceMethod {
    protected function value();
}
?>
--EXPECTF--
interface method HiddenInterfaceMethod::value must be public
