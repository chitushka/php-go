--TEST--
P2 interface member redeclaration is rejected
--FILE--
<?php
interface DuplicateInterfaceMethod {
    public function value();
    public function VALUE();
}
?>
--EXPECTF--
cannot redeclare method DuplicateInterfaceMethod::value
