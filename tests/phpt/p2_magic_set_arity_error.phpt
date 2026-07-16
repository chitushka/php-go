--TEST--
P2 __set must take two arguments
--FILE--
<?php
class BadSetMagicArgs {
    public function __set($name) {}
}
?>
--EXPECTF--
magic method BadSetMagicArgs::__set must take exactly 2 arguments
