<?php
class Box {
    function __construct($value) {
        $this->value = $value;
    }

    function double() {
        return $this->value * 2;
    }
}

$box = new Box(7);
echo $box->double();
