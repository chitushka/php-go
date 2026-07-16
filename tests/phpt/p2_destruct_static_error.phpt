--TEST--
P2 __destruct must not be static
--FILE--
<?php
class StaticDestructMagic {
    public static function __destruct() {}
}
?>
--EXPECTF--
magic method StaticDestructMagic::__destruct must not be static
