--TEST--
P2 method override compatibility
--FILE--
<?php
class Base {
    protected function value() {
        return 1;
    }
}

class Child extends Base {
    public function value() {
        return 2;
    }
}

$child = new Child();
echo $child->value();
?>
--EXPECT--
2
