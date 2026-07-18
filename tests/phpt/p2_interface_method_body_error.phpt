--TEST--
P2 interface methods cannot contain body
--FILE--
<?php
interface BodyInterface {
    public function value() {
        return 1;
    }
}
?>
--EXPECTF--
parse errors: [Interface function BodyInterface::value() cannot contain body]
