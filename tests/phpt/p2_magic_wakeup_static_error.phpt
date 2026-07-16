--TEST--
P2 __wakeup must not be static
--FILE--
<?php
class StaticWakeupMagic {
    public static function __wakeup() {}
}
?>
--EXPECTF--
magic method StaticWakeupMagic::__wakeup must not be static
