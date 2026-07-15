--TEST--
P2 dynamic class names in new
--FILE--
<?php
class Box {
    function __construct($value) {
        $this->value = $value;
    }

    function get() {
        return $this->value;
    }
}

$class = 'Box';
$box = new $class(8);
echo $box->get();
echo "\n";

$holder = 'class';
$other = new $$holder(9);
echo $other->get();
?>
--EXPECT--
8
9
