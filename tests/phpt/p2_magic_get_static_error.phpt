--TEST--
P2 __get must not be static
--FILE--
<?php
class StaticGetMagic {
    public static function __get($name) {}
}
?>
--EXPECTF--
magic method StaticGetMagic::__get must not be static
