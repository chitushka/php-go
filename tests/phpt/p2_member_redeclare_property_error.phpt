--TEST--
P2 class property redeclaration is rejected
--FILE--
<?php
class DuplicateProperty {
    public $value;
    private $value;
}
?>
--EXPECTF--
cannot redeclare property DuplicateProperty::$value
