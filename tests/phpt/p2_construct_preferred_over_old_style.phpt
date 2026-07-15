--TEST--
P2 __construct preferred over old-style constructor
--FILE--
<?php
class Box {
    function Box($value) {
        $this->value = 'old';
    }

    function __construct($value) {
        $this->value = $value;
    }

    function get() {
        return $this->value;
    }
}

$box = new Box('new');
echo $box->get();
?>
--EXPECT--
new
