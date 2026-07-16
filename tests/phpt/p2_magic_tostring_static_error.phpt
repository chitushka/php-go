--TEST--
P2 __toString must not be static
--FILE--
<?php
class StaticToStringMagic {
    public static function __toString() {}
}
?>
--EXPECTF--
magic method StaticToStringMagic::__toString must not be static
