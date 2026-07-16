--TEST--
P2 __callStatic must take two arguments
--FILE--
<?php
class BadCallStaticMagicArgs {
    public static function __callStatic($name) {}
}
?>
--EXPECTF--
magic method BadCallStaticMagicArgs::__callStatic must take exactly 2 arguments
