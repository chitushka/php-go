--TEST--
P2 __set must be public
--FILE--
<?php
class PrivateSetMagic {
    private function __set($name, $value) {}
}
?>
--EXPECTF--
magic method PrivateSetMagic::__set must be public
