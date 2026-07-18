--TEST--
P2 class constructor override signatures are not checked
--FILE--
<?php
class Base {
    public function __construct($value, &$ref) {
        $this->value = $value;
    }
}

class Child extends Base {
    public function __construct() {
        $this->value = 'child';
    }
}

$child = new Child();
echo $child->value;
?>
--EXPECT--
child
