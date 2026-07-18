--TEST--
P2 inherited method parameter type hints must match
--FILE--
<?php
class SignatureBaseType {}
class SignatureOtherType {}
class SignatureParent {
    public function value(SignatureBaseType $value) {}
}
class SignatureChild extends SignatureParent {
    public function value(SignatureOtherType $value) {}
}
?>
--EXPECTF--
method SignatureChild::value parameter 1 type hint must match SignatureParent::value
