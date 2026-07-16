--TEST--
P2 __call must not be static
--FILE--
<?php
class StaticCallMagic {
    public static function __call($name, $args) {}
}
?>
--EXPECTF--
magic method StaticCallMagic::__call must not be static
