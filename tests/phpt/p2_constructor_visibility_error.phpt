--TEST--
P2 private constructor cannot be called from outside class
--FILE--
<?php
class HiddenConstructor {
    private function __construct() {}
}

$hidden = new HiddenConstructor();
?>
--EXPECTF--
cannot access private method HiddenConstructor::__construct
