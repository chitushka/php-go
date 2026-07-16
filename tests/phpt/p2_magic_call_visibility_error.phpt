--TEST--
P2 __call must be public
--FILE--
<?php
class PrivateCallMagic {
    private function __call($name, $args) {}
}
?>
--EXPECTF--
magic method PrivateCallMagic::__call must be public
