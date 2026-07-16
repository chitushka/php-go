--TEST--
P2 abstract methods cannot be private
--FILE--
<?php
abstract class HiddenAbstract {
    abstract private function value();
}
?>
--EXPECTF--
abstract method HiddenAbstract::value cannot be private
