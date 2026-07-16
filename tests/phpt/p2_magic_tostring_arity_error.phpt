--TEST--
P2 __toString must take no arguments
--FILE--
<?php
class BadToStringMagicArgs {
    public function __toString($name) {}
}
?>
--EXPECTF--
magic method BadToStringMagicArgs::__toString must take exactly 0 arguments
