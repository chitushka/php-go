--TEST--
P2 private __clone cannot be called from outside class
--FILE--
<?php
class Box {
    private function __clone() {}
}

$a = new Box();
$b = clone $a;
--EXPECTF--
cannot access private method Box::__clone
