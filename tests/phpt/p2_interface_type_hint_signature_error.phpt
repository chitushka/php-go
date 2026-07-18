--TEST--
P2 interface method parameter type hints must match
--FILE--
<?php
class InterfaceBaseType {}
class InterfaceOtherType {}
interface InterfaceTyped {
    public function value(InterfaceBaseType $value);
}
class InterfaceUser implements InterfaceTyped {
    public function value(InterfaceOtherType $value) {}
}
?>
--EXPECTF--
method InterfaceUser::value parameter 1 type hint must match InterfaceTyped::value
