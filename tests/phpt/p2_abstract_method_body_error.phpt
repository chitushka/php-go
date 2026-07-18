--TEST--
P2 abstract methods cannot contain body
--FILE--
<?php
abstract class BodyAbstract {
    abstract public function value() {
        return 1;
    }
}
?>
--EXPECTF--
parse errors: [Abstract function BodyAbstract::value() cannot contain body]
