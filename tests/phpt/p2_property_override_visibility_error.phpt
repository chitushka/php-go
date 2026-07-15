--TEST--
P2 inherited property visibility cannot be reduced
--FILE--
<?php
class Base {
    public $value = 1;
}

class Child extends Base {
    protected $value = 2;
}
?>
--EXPECTF--
cannot reduce visibility of inherited property Base::$value in Child
