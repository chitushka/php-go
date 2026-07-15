--TEST--
P2 __toString must return string
--FILE--
<?php
class Label {
    public function __toString() {
        return 123;
    }
}

$label = new Label();
echo $label;
--EXPECTF--
__toString must return string
