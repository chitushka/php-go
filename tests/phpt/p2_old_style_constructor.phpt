--TEST--
P2 old-style constructor
--FILE--
<?php
class Box {
    function Box($value) {
        $this->value = $value;
    }

    function get() {
        return $this->value;
    }
}

$box = new Box(9);
echo $box->get();
?>
--EXPECT--
9
