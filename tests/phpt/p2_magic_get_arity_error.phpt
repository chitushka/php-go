--TEST--
P2 __get must take one argument
--FILE--
<?php
class BadGetMagicArgs {
    public function __get() {}
}
?>
--EXPECTF--
magic method BadGetMagicArgs::__get must take exactly 1 arguments
