--TEST--
P2 class method redeclaration is rejected
--FILE--
<?php
class DuplicateMethod {
    public function value() {}
    public function VALUE() {}
}
?>
--EXPECTF--
cannot redeclare method DuplicateMethod::value
