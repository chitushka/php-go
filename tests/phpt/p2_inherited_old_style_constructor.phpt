--TEST--
P2 inherited old-style constructor
--FILE--
<?php
class Base {
    function Base($value) {
        $this->value = $value;
    }
}

class Child extends Base {
    function get() {
        return $this->value;
    }
}

$child = new Child(7);
echo $child->get();
?>
--EXPECT--
7
