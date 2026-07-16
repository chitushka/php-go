--TEST--
P2 __toString must be public
--FILE--
<?php
class PrivateToStringMagic {
    private function __toString() {}
}
?>
--EXPECTF--
magic method PrivateToStringMagic::__toString must be public
