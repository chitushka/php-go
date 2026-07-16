--TEST--
P2 __sleep must take no arguments
--FILE--
<?php
class BadSleepMagicArgs {
    public function __sleep($name) {}
}
?>
--EXPECTF--
magic method BadSleepMagicArgs::__sleep must take exactly 0 arguments
