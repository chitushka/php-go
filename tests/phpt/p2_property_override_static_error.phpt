--TEST--
P2 inherited property static modifier cannot change
--FILE--
<?php
class Base {
    public $value = 1;
}

class Child extends Base {
    public static $value = 2;
}
?>
--EXPECTF--
cannot change static modifier of inherited property Base::$value in Child
