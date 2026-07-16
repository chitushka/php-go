--TEST--
P2 __clone must not be static
--FILE--
<?php
class StaticCloneMagic {
    public static function __clone() {}
}
?>
--EXPECTF--
magic method StaticCloneMagic::__clone must not be static
