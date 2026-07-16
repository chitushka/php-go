--TEST--
P2 __get must be public
--FILE--
<?php
class PrivateGetMagic {
    private function __get($name) {}
}
?>
--EXPECTF--
magic method PrivateGetMagic::__get must be public
