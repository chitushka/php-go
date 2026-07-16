--TEST--
P2 __invoke must be public
--FILE--
<?php
class PrivateInvokeMagic {
    private function __invoke() {}
}
?>
--EXPECTF--
magic method PrivateInvokeMagic::__invoke must be public
