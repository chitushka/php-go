--TEST--
P2 __clone must take no arguments
--FILE--
<?php
class BadCloneMagicArgs {
    private function __clone($name) {}
}
?>
--EXPECTF--
magic method BadCloneMagicArgs::__clone must take exactly 0 arguments
