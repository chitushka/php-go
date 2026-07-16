--TEST--
P2 __sleep must be public
--FILE--
<?php
class PrivateSleepMagic {
    private function __sleep() {}
}
?>
--EXPECTF--
magic method PrivateSleepMagic::__sleep must be public
