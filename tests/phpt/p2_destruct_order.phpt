--TEST--
P2 __destruct runs in reverse creation order
--FILE--
<?php
class Item {
    function __construct($name) {
        $this->name = $name;
    }

    function __destruct() {
        echo $this->name;
    }
}

$a = new Item('a');
$b = new Item('b');
echo "body";
?>
--EXPECT--
bodyba
