--TEST--
P2 __destruct must take no arguments
--FILE--
<?php
class BadDestructMagicArgs {
    public function __destruct($name) {}
}
?>
--EXPECTF--
magic method BadDestructMagicArgs::__destruct must take exactly 0 arguments
