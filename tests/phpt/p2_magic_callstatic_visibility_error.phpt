--TEST--
P2 __callStatic must be public
--FILE--
<?php
class PrivateCallStaticMagic {
    private static function __callStatic($name, $args) {}
}
?>
--EXPECTF--
magic method PrivateCallStaticMagic::__callStatic must be public
