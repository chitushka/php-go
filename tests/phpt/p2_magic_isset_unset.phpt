--TEST--
P2 magic __isset and __unset
--FILE--
<?php
class Bag {
    private $items;

    public function __construct() {
        $this->items = array('yes' => 'value', 'zero' => '0');
    }

    public function __isset($name) {
        return isset($this->items[$name]);
    }

    public function __get($name) {
        return $this->items[$name];
    }

    public function __unset($name) {
        unset($this->items[$name]);
        echo 'unset:' . $name . "\n";
    }
}

$bag = new Bag();
if (isset($bag->yes)) { echo "yes\n"; }
if (!isset($bag->missing)) { echo "missing\n"; }
if (empty($bag->zero)) { echo "zero\n"; }
unset($bag->yes);
if (!isset($bag->yes)) { echo "gone\n"; }
?>
--EXPECT--
yes
missing
zero
unset:yes
gone
