--TEST--
P2 __invoke must not be static
--FILE--
<?php
class StaticInvokeMagic {
    public static function __invoke() {}
}
?>
--EXPECTF--
magic method StaticInvokeMagic::__invoke must not be static
