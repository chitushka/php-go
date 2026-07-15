--TEST--
P2 method override cannot add required parameters
--FILE--
<?php
class Base {
    public function label($name) {
        return $name;
    }
}

class Child extends Base {
    public function label($name, $suffix) {
        return $name . $suffix;
    }
}
?>
--EXPECTF--
method Child::label cannot require 2 parameters to match Base::label requiring 1
