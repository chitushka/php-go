--TEST--
P2 inherited method visibility cannot be reduced
--FILE--
<?php
class Base {
    public function value() {
        return 1;
    }
}

class Child extends Base {
    protected function value() {
        return 2;
    }
}
?>
--EXPECTF--
cannot reduce visibility of inherited method Base::value in Child
