--TEST--
P2 __construct must not be static
--FILE--
<?php
class StaticConstructMagic {
    public static function __construct() {}
}
?>
--EXPECTF--
magic method StaticConstructMagic::__construct must not be static
