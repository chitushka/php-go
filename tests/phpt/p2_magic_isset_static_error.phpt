--TEST--
P2 __isset must not be static
--FILE--
<?php
class StaticIssetMagic {
    public static function __isset($name) {}
}
?>
--EXPECTF--
magic method StaticIssetMagic::__isset must not be static
