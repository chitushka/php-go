--TEST--
P2 __call must take two arguments
--FILE--
<?php
class BadCallMagicArgs {
    public function __call($name) {}
}
?>
--EXPECTF--
magic method BadCallMagicArgs::__call must take exactly 2 arguments
