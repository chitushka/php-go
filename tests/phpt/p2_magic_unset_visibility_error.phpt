--TEST--
P2 __unset must be public
--FILE--
<?php
class PrivateUnsetMagic {
    private function __unset($name) {}
}
?>
--EXPECTF--
magic method PrivateUnsetMagic::__unset must be public
